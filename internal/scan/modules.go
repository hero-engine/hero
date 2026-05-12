package scan

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Subproject represents a detected subproject/module within a multi-module build.
type Subproject struct {
	Name     string // e.g. "morpheus-core", "packages/web"
	Dir      string // relative directory path
	Language string // primary language if detectable
	Kind     string // "grails-plugin", "gradle-module", "cargo-member", "go-module", "npm-workspace", etc.
}

// MultiModuleInfo holds details about multi-module project structure.
type MultiModuleInfo struct {
	BuildSystem string       // "gradle", "cargo-workspace", "go-workspace", "npm-workspace", "maven"
	RootConfig  string       // settings.gradle, Cargo.toml, go.work, etc.
	Subprojects []Subproject // detected subprojects
}

// FrameworkDetail holds deeper framework-specific information.
type FrameworkDetail struct {
	Name        string            // "Grails", "Spring Boot", "Django", "Rails", etc.
	Version     string            // detected version if available
	Conventions []string          // detected convention patterns (e.g. "grails-app/ structure", "MVC")
	Directories []string          // key framework directories
	ConfigFiles []string          // framework config files found
	Metadata    map[string]string // additional framework-specific info
}

// DetectMultiModule performs deep analysis of multi-module project structure.
// It reads build configuration files to discover subprojects rather than just
// checking for directory naming patterns.
func DetectMultiModule(projectRoot string) *MultiModuleInfo {
	// Try each build system in order of specificity

	// 1. Gradle (settings.gradle / settings.gradle.kts)
	if info := detectGradleModules(projectRoot); info != nil {
		return info
	}

	// 2. Maven (pom.xml with modules)
	if info := detectMavenModules(projectRoot); info != nil {
		return info
	}

	// 3. Cargo workspace (Cargo.toml with [workspace])
	if info := detectCargoWorkspace(projectRoot); info != nil {
		return info
	}

	// 4. Go workspace (go.work)
	if info := detectGoWorkspace(projectRoot); info != nil {
		return info
	}

	// 5. NPM/pnpm workspaces
	if info := detectNPMWorkspace(projectRoot); info != nil {
		return info
	}

	return nil
}

// detectGradleModules reads settings.gradle or settings.gradle.kts to find subprojects.
func detectGradleModules(root string) *MultiModuleInfo {
	var settingsFile, content string

	for _, name := range []string{"settings.gradle", "settings.gradle.kts"} {
		full := filepath.Join(root, name)
		data, err := os.ReadFile(full)
		if err == nil {
			settingsFile = name
			content = string(data)
			break
		}
	}
	if settingsFile == "" {
		return nil
	}

	subprojects := parseGradleSettings(content)
	if len(subprojects) == 0 {
		return nil
	}

	// Enrich subprojects with language/kind detection
	for i := range subprojects {
		enrichGradleSubproject(root, &subprojects[i])
	}

	return &MultiModuleInfo{
		BuildSystem: "gradle",
		RootConfig:  settingsFile,
		Subprojects: subprojects,
	}
}

// parseGradleSettings extracts subproject names from settings.gradle content.
// Handles both include ':project' and include(':project') syntaxes,
// as well as multi-line include blocks.
func parseGradleSettings(content string) []Subproject {
	var subprojects []Subproject
	seen := map[string]bool{}

	// Match include ':name' or include(':name') or include ':name1', ':name2'
	// Also handle include(':name1', ':name2')
	reInclude := regexp.MustCompile(`include\s*\(?([^)\n]+)\)?`)
	reProject := regexp.MustCompile(`'([^']+)'|"([^"]+)"`)

	for _, match := range reInclude.FindAllStringSubmatch(content, -1) {
		args := match[1]
		for _, pm := range reProject.FindAllStringSubmatch(args, -1) {
			name := pm[1]
			if name == "" {
				name = pm[2]
			}
			// Remove leading colon
			name = strings.TrimPrefix(name, ":")
			if name != "" && !seen[name] {
				seen[name] = true
				// Convert Gradle ':a:b' notation to directory path
				dir := strings.ReplaceAll(name, ":", "/")
				subprojects = append(subprojects, Subproject{
					Name: name,
					Dir:  dir,
					Kind: "gradle-module",
				})
			}
		}
	}

	// Also check for project(':name').projectDir = file('path') overrides
	reProjDir := regexp.MustCompile(`project\(['"]([^'"]+)['"]\)\.projectDir\s*=\s*(?:file\(['"]([^'"]+)['"]\)|new\s+File\(['"]([^'"]+)['"]\))`)
	for _, match := range reProjDir.FindAllStringSubmatch(content, -1) {
		name := strings.TrimPrefix(match[1], ":")
		dir := match[2]
		if dir == "" {
			dir = match[3]
		}
		// Update existing subproject's directory
		for i := range subprojects {
			if subprojects[i].Name == name {
				subprojects[i].Dir = dir
				break
			}
		}
	}

	return subprojects
}

// enrichGradleSubproject detects language and kind for a Gradle subproject.
func enrichGradleSubproject(root string, sp *Subproject) {
	dir := filepath.Join(root, sp.Dir)
	if _, err := os.Stat(dir); err != nil {
		return
	}

	// Check for Grails plugin structure
	if dirExists(dir, "grails-app") {
		sp.Kind = "grails-plugin"
		sp.Language = "Groovy"
		return
	}

	// Check for Kotlin source
	if dirExists(dir, "src/main/kotlin") {
		sp.Language = "Kotlin"
		return
	}

	// Check for Java source
	if dirExists(dir, "src/main/java") {
		sp.Language = "Java"
		return
	}

	// Check for Groovy source
	if dirExists(dir, "src/main/groovy") {
		sp.Language = "Groovy"
		return
	}

	// Check build.gradle for plugin hints
	for _, buildFile := range []string{"build.gradle", "build.gradle.kts"} {
		data, err := os.ReadFile(filepath.Join(dir, buildFile))
		if err != nil {
			continue
		}
		buildContent := string(data)
		if strings.Contains(buildContent, "grails-plugin") || strings.Contains(buildContent, "org.grails") {
			sp.Kind = "grails-plugin"
			sp.Language = "Groovy"
		} else if strings.Contains(buildContent, "kotlin") || strings.Contains(buildContent, "org.jetbrains.kotlin") {
			sp.Language = "Kotlin"
		} else if strings.Contains(buildContent, "groovy") {
			sp.Language = "Groovy"
		} else if strings.Contains(buildContent, "java") {
			sp.Language = "Java"
		}
		return
	}
}

// detectMavenModules reads pom.xml to find <modules> declarations.
func detectMavenModules(root string) *MultiModuleInfo {
	data, err := os.ReadFile(filepath.Join(root, "pom.xml"))
	if err != nil {
		return nil
	}
	content := string(data)

	// Extract module names from <modules><module>name</module></modules>
	reModule := regexp.MustCompile(`<module>([^<]+)</module>`)
	matches := reModule.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	var subprojects []Subproject
	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		subprojects = append(subprojects, Subproject{
			Name: name,
			Dir:  name,
			Kind: "maven-module",
		})
	}

	// Detect language for each module
	for i := range subprojects {
		dir := filepath.Join(root, subprojects[i].Dir)
		if dirExists(dir, "src/main/kotlin") {
			subprojects[i].Language = "Kotlin"
		} else if dirExists(dir, "src/main/java") {
			subprojects[i].Language = "Java"
		} else if dirExists(dir, "src/main/groovy") {
			subprojects[i].Language = "Groovy"
		}
	}

	return &MultiModuleInfo{
		BuildSystem: "maven",
		RootConfig:  "pom.xml",
		Subprojects: subprojects,
	}
}

// detectCargoWorkspace reads Cargo.toml for [workspace] members.
func detectCargoWorkspace(root string) *MultiModuleInfo {
	data, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil {
		return nil
	}
	content := string(data)

	if !strings.Contains(content, "[workspace]") {
		return nil
	}

	// Extract members from members = ["crate1", "crate2"]
	reMembers := regexp.MustCompile(`members\s*=\s*\[([^\]]+)\]`)
	m := reMembers.FindStringSubmatch(content)
	if m == nil {
		return nil
	}

	var subprojects []Subproject
	reName := regexp.MustCompile(`"([^"]+)"`)
	for _, nm := range reName.FindAllStringSubmatch(m[1], -1) {
		name := nm[1]
		// Expand glob patterns like "crates/*"
		if strings.Contains(name, "*") {
			expanded := expandGlob(root, name)
			for _, dir := range expanded {
				base := filepath.Base(dir)
				subprojects = append(subprojects, Subproject{
					Name:     base,
					Dir:      dir,
					Language: "Rust",
					Kind:     "cargo-member",
				})
			}
		} else {
			subprojects = append(subprojects, Subproject{
				Name:     filepath.Base(name),
				Dir:      name,
				Language: "Rust",
				Kind:     "cargo-member",
			})
		}
	}

	if len(subprojects) == 0 {
		return nil
	}

	return &MultiModuleInfo{
		BuildSystem: "cargo-workspace",
		RootConfig:  "Cargo.toml",
		Subprojects: subprojects,
	}
}

// detectGoWorkspace reads go.work for use directives.
func detectGoWorkspace(root string) *MultiModuleInfo {
	data, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		return nil
	}
	content := string(data)

	var subprojects []Subproject
	inUse := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "use (") || line == "use (" {
			inUse = true
			continue
		}
		if line == ")" && inUse {
			inUse = false
			continue
		}
		if strings.HasPrefix(line, "use ") && !strings.Contains(line, "(") {
			// Single-line use
			dir := strings.TrimSpace(strings.TrimPrefix(line, "use "))
			dir = strings.Trim(dir, `"`)
			if dir != "" && dir != "." {
				subprojects = append(subprojects, Subproject{
					Name:     filepath.Base(dir),
					Dir:      dir,
					Language: "Go",
					Kind:     "go-module",
				})
			}
			continue
		}
		if inUse && line != "" && !strings.HasPrefix(line, "//") {
			dir := strings.Trim(line, `" `)
			if dir != "" && dir != "." {
				subprojects = append(subprojects, Subproject{
					Name:     filepath.Base(dir),
					Dir:      dir,
					Language: "Go",
					Kind:     "go-module",
				})
			}
		}
	}

	if len(subprojects) == 0 {
		return nil
	}

	return &MultiModuleInfo{
		BuildSystem: "go-workspace",
		RootConfig:  "go.work",
		Subprojects: subprojects,
	}
}

// detectNPMWorkspace detects pnpm-workspace.yaml or package.json workspaces.
func detectNPMWorkspace(root string) *MultiModuleInfo {
	// Check pnpm-workspace.yaml
	if data, err := os.ReadFile(filepath.Join(root, "pnpm-workspace.yaml")); err == nil {
		content := string(data)
		rePkg := regexp.MustCompile(`-\s+['"]?([^'"#\n]+)['"]?`)
		var subprojects []Subproject
		for _, m := range rePkg.FindAllStringSubmatch(content, -1) {
			pattern := strings.TrimSpace(m[1])
			if strings.Contains(pattern, "*") {
				expanded := expandGlob(root, pattern)
				for _, dir := range expanded {
					subprojects = append(subprojects, Subproject{
						Name: filepath.Base(dir),
						Dir:  dir,
						Kind: "npm-workspace",
					})
				}
			} else {
				subprojects = append(subprojects, Subproject{
					Name: filepath.Base(pattern),
					Dir:  pattern,
					Kind: "npm-workspace",
				})
			}
		}
		if len(subprojects) > 0 {
			return &MultiModuleInfo{
				BuildSystem: "pnpm-workspace",
				RootConfig:  "pnpm-workspace.yaml",
				Subprojects: subprojects,
			}
		}
	}

	// Check package.json workspaces
	if data, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
		content := string(data)
		if !strings.Contains(content, `"workspaces"`) {
			return nil
		}

		// Extract workspace patterns
		reWS := regexp.MustCompile(`"workspaces"\s*:\s*\[([^\]]+)\]`)
		m := reWS.FindStringSubmatch(content)
		if m == nil {
			return nil
		}

		rePat := regexp.MustCompile(`"([^"]+)"`)
		var subprojects []Subproject
		for _, pm := range rePat.FindAllStringSubmatch(m[1], -1) {
			pattern := pm[1]
			if strings.Contains(pattern, "*") {
				expanded := expandGlob(root, pattern)
				for _, dir := range expanded {
					subprojects = append(subprojects, Subproject{
						Name: filepath.Base(dir),
						Dir:  dir,
						Kind: "npm-workspace",
					})
				}
			} else {
				subprojects = append(subprojects, Subproject{
					Name: filepath.Base(pattern),
					Dir:  pattern,
					Kind: "npm-workspace",
				})
			}
		}

		if len(subprojects) > 0 {
			return &MultiModuleInfo{
				BuildSystem: "npm-workspace",
				RootConfig:  "package.json",
				Subprojects: subprojects,
			}
		}
	}

	return nil
}

// DetectFrameworkDetails performs deep framework-specific analysis.
func DetectFrameworkDetails(projectRoot string, r *Result) []FrameworkDetail {
	var details []FrameworkDetail

	// Grails detection
	if fd := detectGrailsFramework(projectRoot); fd != nil {
		details = append(details, *fd)
	}

	// Spring Boot detection
	if fd := detectSpringBoot(projectRoot); fd != nil {
		details = append(details, *fd)
	}

	// Django detection
	if fd := detectDjango(projectRoot); fd != nil {
		details = append(details, *fd)
	}

	// Rails detection
	if fd := detectRails(projectRoot); fd != nil {
		details = append(details, *fd)
	}

	// Laravel detection
	if fd := detectLaravel(projectRoot); fd != nil {
		details = append(details, *fd)
	}

	return details
}

// detectGrailsFramework detects Grails project structure and configuration.
func detectGrailsFramework(root string) *FrameworkDetail {
	// Look for grails-app at root or in subprojects
	grailsDirs := findDirsNamed(root, "grails-app", 2) // max depth 2
	if len(grailsDirs) == 0 {
		return nil
	}

	fd := &FrameworkDetail{
		Name:     "Grails",
		Metadata: map[string]string{},
	}

	// Read application.yml or application.groovy for version info
	for _, gDir := range grailsDirs {
		confDir := filepath.Join(gDir, "conf")
		for _, confFile := range []string{"application.yml", "application.groovy", "application.yaml"} {
			full := filepath.Join(confDir, confFile)
			if _, err := os.Stat(full); err == nil {
				fd.ConfigFiles = append(fd.ConfigFiles, confFile)
			}
		}
	}

	// Detect Grails version from build.gradle or gradle.properties
	version := detectGrailsVersion(root)
	if version != "" {
		fd.Version = version
	}

	// Detect conventions from grails-app structure
	for _, gDir := range grailsDirs {
		rel, _ := filepath.Rel(root, filepath.Dir(gDir))
		fd.Directories = append(fd.Directories, rel)

		// Check for standard Grails subdirectories
		stdDirs := []string{"controllers", "services", "domain", "views", "taglib", "assets", "conf", "init", "jobs", "migrations", "websockets"}
		for _, sd := range stdDirs {
			if dirExists(gDir, sd) {
				fd.Conventions = append(fd.Conventions, sd)
			}
		}
	}

	return fd
}

// detectGrailsVersion tries to find the Grails version from build configuration.
func detectGrailsVersion(root string) string {
	// Check gradle.properties
	if data, err := os.ReadFile(filepath.Join(root, "gradle.properties")); err == nil {
		content := string(data)
		re := regexp.MustCompile(`(?m)^grailsVersion\s*=\s*(.+)$`)
		if m := re.FindStringSubmatch(content); m != nil {
			return strings.TrimSpace(m[1])
		}
	}

	// Check build.gradle for grails version in buildscript
	for _, name := range []string{"build.gradle", "build.gradle.kts"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		content := string(data)
		re := regexp.MustCompile(`grails[._-]?[Vv]ersion\s*[=:]\s*['"]([^'"]+)['"]`)
		if m := re.FindStringSubmatch(content); m != nil {
			return m[1]
		}
	}

	return ""
}

// detectSpringBoot detects Spring Boot projects.
func detectSpringBoot(root string) *FrameworkDetail {
	// Check for Spring Boot indicators
	indicators := []string{
		"src/main/resources/application.properties",
		"src/main/resources/application.yml",
		"src/main/resources/application.yaml",
	}

	found := false
	var configFiles []string
	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(root, ind)); err == nil {
			found = true
			configFiles = append(configFiles, ind)
		}
	}

	if !found {
		// Also check build files for spring boot plugin
		for _, name := range []string{"build.gradle", "build.gradle.kts", "pom.xml"} {
			data, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				continue
			}
			if strings.Contains(string(data), "spring-boot") || strings.Contains(string(data), "org.springframework.boot") {
				found = true
				break
			}
		}
	}

	if !found {
		return nil
	}

	// Don't double-detect if this is also a Grails project (Grails uses Spring Boot underneath)
	grailsDirs := findDirsNamed(root, "grails-app", 2)
	if len(grailsDirs) > 0 {
		return nil
	}

	fd := &FrameworkDetail{
		Name:        "Spring Boot",
		ConfigFiles: configFiles,
		Metadata:    map[string]string{},
	}

	// Read application.properties/yml for port, profile info
	for _, cf := range configFiles {
		data, err := os.ReadFile(filepath.Join(root, cf))
		if err != nil {
			continue
		}
		content := string(data)
		if strings.Contains(content, "server.port") {
			re := regexp.MustCompile(`server\.port\s*[=:]\s*(\d+)`)
			if m := re.FindStringSubmatch(content); m != nil {
				fd.Metadata["server-port"] = m[1]
			}
		}
	}

	return fd
}

// detectDjango detects Django projects.
func detectDjango(root string) *FrameworkDetail {
	if !fileExists(root, "manage.py") {
		return nil
	}

	// Verify it's Django (not just any manage.py)
	content := readFileStr(root, "manage.py")
	if !strings.Contains(content, "django") && !strings.Contains(content, "DJANGO") {
		return nil
	}

	fd := &FrameworkDetail{
		Name:     "Django",
		Metadata: map[string]string{},
	}

	// Find settings module
	re := regexp.MustCompile(`DJANGO_SETTINGS_MODULE.*['"]([\w.]+)['"]`)
	if m := re.FindStringSubmatch(content); m != nil {
		fd.Metadata["settings-module"] = m[1]
	}

	// Detect Django apps (directories with apps.py or models.py)
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		appDir := filepath.Join(root, e.Name())
		if fileExists(appDir, "apps.py") || (fileExists(appDir, "models.py") && fileExists(appDir, "views.py")) {
			fd.Conventions = append(fd.Conventions, e.Name())
		}
	}

	return fd
}

// detectRails detects Ruby on Rails projects.
func detectRails(root string) *FrameworkDetail {
	if !fileExists(root, "config/routes.rb") {
		return nil
	}

	fd := &FrameworkDetail{
		Name:     "Ruby on Rails",
		Metadata: map[string]string{},
	}

	// Check for standard Rails directories
	railsDirs := []string{"app/models", "app/controllers", "app/views", "app/helpers", "app/mailers", "app/jobs", "db/migrate", "config/initializers"}
	for _, d := range railsDirs {
		if _, err := os.Stat(filepath.Join(root, d)); err == nil {
			fd.Directories = append(fd.Directories, d)
		}
	}

	// Detect Rails version from Gemfile.lock
	if data, err := os.ReadFile(filepath.Join(root, "Gemfile.lock")); err == nil {
		re := regexp.MustCompile(`rails\s+\((\d+\.\d+\.\d+)\)`)
		if m := re.FindStringSubmatch(string(data)); m != nil {
			fd.Version = m[1]
		}
	}

	return fd
}

// detectLaravel detects Laravel projects.
func detectLaravel(root string) *FrameworkDetail {
	if !fileExists(root, "artisan") {
		return nil
	}

	content := readFileStr(root, "artisan")
	if !strings.Contains(content, "laravel") && !strings.Contains(content, "Laravel") {
		return nil
	}

	fd := &FrameworkDetail{
		Name:     "Laravel",
		Metadata: map[string]string{},
	}

	laravelDirs := []string{"app/Http/Controllers", "app/Models", "database/migrations", "resources/views", "routes"}
	for _, d := range laravelDirs {
		if _, err := os.Stat(filepath.Join(root, d)); err == nil {
			fd.Directories = append(fd.Directories, d)
		}
	}

	return fd
}

// Helpers

func dirExists(root, name string) bool {
	info, err := os.Stat(filepath.Join(root, name))
	return err == nil && info.IsDir()
}

// findDirsNamed finds directories with the given name up to maxDepth levels deep.
func findDirsNamed(root, name string, maxDepth int) []string {
	var found []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		depth := strings.Count(rel, string(os.PathSeparator))
		if depth > maxDepth {
			return filepath.SkipDir
		}
		// Skip hidden dirs and common skip dirs
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" || base == "build" || base == "target" {
			return filepath.SkipDir
		}
		if info.Name() == name {
			found = append(found, path)
		}
		return nil
	})
	return found
}

// expandGlob expands a glob pattern relative to root, returning relative paths
// to directories that match.
func expandGlob(root, pattern string) []string {
	full := filepath.Join(root, pattern)
	matches, err := filepath.Glob(full)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err == nil && info.IsDir() {
			rel, _ := filepath.Rel(root, m)
			dirs = append(dirs, rel)
		}
	}
	return dirs
}
