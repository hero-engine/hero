package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectGradleModules(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"settings.gradle": `
rootProject.name = 'morpheus-lts'

include 'morpheus-ui'
include 'morpheus-core'
include 'morpheus-domains'
include 'morpheus-api'
include 'clouds:vmware'
include 'clouds:amazon'
include 'backups:veeam'
include 'utils:common'
`,
		"build.gradle":                          "// root build\n",
		"morpheus-ui/grails-app/conf/empty.txt": "",
		"morpheus-core/src/main/groovy/x.groovy": "class X {}",
		"morpheus-domains/src/main/groovy/y.groovy": "class Y {}",
		"clouds/vmware/grails-app/services/empty.txt": "",
		"utils/common/src/main/groovy/u.groovy": "class U {}",
	})

	info := detectGradleModules(dir)
	if info == nil {
		t.Fatal("expected gradle modules, got nil")
	}

	if info.BuildSystem != "gradle" {
		t.Errorf("BuildSystem = %q, want gradle", info.BuildSystem)
	}

	names := map[string]bool{}
	for _, sp := range info.Subprojects {
		names[sp.Name] = true
	}

	expected := []string{"morpheus-ui", "morpheus-core", "morpheus-domains", "morpheus-api", "clouds:vmware", "clouds:amazon", "backups:veeam", "utils:common"}
	for _, e := range expected {
		cleanName := strings.ReplaceAll(e, ":", "/") // parseGradleSettings stores as path separator
		// Check either format
		if !names[e] && !names[cleanName] {
			// The name stored depends on whether colon is replaced
			found := false
			for n := range names {
				if strings.HasSuffix(n, filepath.Base(e)) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("missing subproject %q in %v", e, names)
			}
		}
	}

	// Check that Grails subprojects are detected
	for _, sp := range info.Subprojects {
		if sp.Name == "morpheus-ui" && sp.Kind != "grails-plugin" {
			t.Errorf("morpheus-ui Kind = %q, want grails-plugin", sp.Kind)
		}
	}
}

func TestDetectGradleModulesKotlinDSL(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"settings.gradle.kts": `
rootProject.name = "my-project"
include(":app")
include(":lib")
include(":core", ":api")
`,
		"app/src/main/kotlin/App.kt": "fun main() {}",
		"lib/src/main/java/Lib.java": "class Lib {}",
	})

	info := detectGradleModules(dir)
	if info == nil {
		t.Fatal("expected gradle modules from .kts, got nil")
	}

	if len(info.Subprojects) < 4 {
		t.Errorf("expected at least 4 subprojects, got %d", len(info.Subprojects))
	}

	// Check language detection
	for _, sp := range info.Subprojects {
		if sp.Name == "app" && sp.Language != "Kotlin" {
			t.Errorf("app Language = %q, want Kotlin", sp.Language)
		}
		if sp.Name == "lib" && sp.Language != "Java" {
			t.Errorf("lib Language = %q, want Java", sp.Language)
		}
	}
}

func TestDetectGradleModulesNone(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"build.gradle": "// just a single module project\n",
	})

	info := detectGradleModules(dir)
	if info != nil {
		t.Error("expected nil for project without settings.gradle")
	}
}

func TestParseGradleSettings(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int // expected number of subprojects
	}{
		{
			name: "simple includes",
			content: `include 'app'
include 'lib'`,
			want: 2,
		},
		{
			name: "kotlin DSL includes",
			content: `include(":app")
include(":lib", ":core")`,
			want: 3,
		},
		{
			name:    "nested paths",
			content: `include 'clouds:vmware', 'clouds:amazon'`,
			want:    2,
		},
		{
			name:    "empty",
			content: `rootProject.name = 'test'`,
			want:    0,
		},
		{
			name: "double-quoted",
			content: `include "app"
include "lib"`,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGradleSettings(tt.content)
			if len(got) != tt.want {
				names := make([]string, len(got))
				for i, sp := range got {
					names[i] = sp.Name
				}
				t.Errorf("parseGradleSettings() got %d subprojects %v, want %d", len(got), names, tt.want)
			}
		})
	}
}

func TestDetectMavenModules(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"pom.xml": `<?xml version="1.0"?>
<project>
  <groupId>com.example</groupId>
  <artifactId>parent</artifactId>
  <packaging>pom</packaging>
  <modules>
    <module>core</module>
    <module>api</module>
    <module>web</module>
  </modules>
</project>`,
		"core/src/main/java/Core.java": "class Core {}",
		"api/src/main/java/Api.java":   "class Api {}",
	})

	info := detectMavenModules(dir)
	if info == nil {
		t.Fatal("expected maven modules, got nil")
	}

	if info.BuildSystem != "maven" {
		t.Errorf("BuildSystem = %q, want maven", info.BuildSystem)
	}
	if len(info.Subprojects) != 3 {
		t.Errorf("expected 3 subprojects, got %d", len(info.Subprojects))
	}

	// Check Java detection
	for _, sp := range info.Subprojects {
		if sp.Name == "core" && sp.Language != "Java" {
			t.Errorf("core Language = %q, want Java", sp.Language)
		}
	}
}

func TestDetectMavenModulesNone(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"pom.xml": `<?xml version="1.0"?><project><groupId>com.example</groupId></project>`,
	})

	info := detectMavenModules(dir)
	if info != nil {
		t.Error("expected nil for single-module Maven project")
	}
}

func TestDetectCargoWorkspace(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"Cargo.toml": `
[workspace]
members = ["core", "cli", "lib"]
`,
		"core/Cargo.toml": `[package]\nname = "core"`,
		"cli/Cargo.toml":  `[package]\nname = "cli"`,
		"lib/Cargo.toml":  `[package]\nname = "lib"`,
	})

	info := detectCargoWorkspace(dir)
	if info == nil {
		t.Fatal("expected cargo workspace, got nil")
	}

	if info.BuildSystem != "cargo-workspace" {
		t.Errorf("BuildSystem = %q, want cargo-workspace", info.BuildSystem)
	}
	if len(info.Subprojects) != 3 {
		t.Errorf("expected 3 subprojects, got %d", len(info.Subprojects))
	}
	for _, sp := range info.Subprojects {
		if sp.Language != "Rust" {
			t.Errorf("Language = %q, want Rust", sp.Language)
		}
	}
}

func TestDetectCargoWorkspaceGlob(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"Cargo.toml": `
[workspace]
members = ["crates/*"]
`,
		"crates/core/Cargo.toml": `[package]\nname = "core"`,
		"crates/cli/Cargo.toml":  `[package]\nname = "cli"`,
	})

	info := detectCargoWorkspace(dir)
	if info == nil {
		t.Fatal("expected cargo workspace with glob, got nil")
	}
	if len(info.Subprojects) != 2 {
		t.Errorf("expected 2 subprojects from glob, got %d", len(info.Subprojects))
	}
}

func TestDetectCargoWorkspaceNone(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"Cargo.toml": `[package]\nname = "single"`,
	})

	info := detectCargoWorkspace(dir)
	if info != nil {
		t.Error("expected nil for single-crate project")
	}
}

func TestDetectGoWorkspace(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"go.work": `go 1.22

use (
	./core
	./api
	./cmd/server
)
`,
		"core/go.mod":       "module example.com/core\n",
		"api/go.mod":        "module example.com/api\n",
		"cmd/server/go.mod": "module example.com/cmd/server\n",
	})

	info := detectGoWorkspace(dir)
	if info == nil {
		t.Fatal("expected go workspace, got nil")
	}

	if info.BuildSystem != "go-workspace" {
		t.Errorf("BuildSystem = %q, want go-workspace", info.BuildSystem)
	}
	if len(info.Subprojects) != 3 {
		t.Errorf("expected 3 subprojects, got %d", len(info.Subprojects))
	}
}

func TestDetectGoWorkspaceSingleUse(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"go.work":    "go 1.22\n\nuse ./mymod\n",
		"mymod/go.mod": "module example.com/mymod\n",
	})

	info := detectGoWorkspace(dir)
	if info == nil {
		t.Fatal("expected go workspace, got nil")
	}
	if len(info.Subprojects) != 1 {
		t.Errorf("expected 1 subproject, got %d", len(info.Subprojects))
	}
}

func TestDetectNPMWorkspacePnpm(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"pnpm-workspace.yaml": `packages:
  - 'packages/*'
  - 'apps/*'
`,
		"packages/ui/package.json":  `{"name":"ui"}`,
		"packages/lib/package.json": `{"name":"lib"}`,
		"apps/web/package.json":     `{"name":"web"}`,
	})

	info := detectNPMWorkspace(dir)
	if info == nil {
		t.Fatal("expected pnpm workspace, got nil")
	}
	if info.BuildSystem != "pnpm-workspace" {
		t.Errorf("BuildSystem = %q, want pnpm-workspace", info.BuildSystem)
	}
	if len(info.Subprojects) < 3 {
		t.Errorf("expected at least 3 subprojects, got %d", len(info.Subprojects))
	}
}

func TestDetectNPMWorkspacePackageJSON(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"package.json": `{
  "name": "monorepo",
  "workspaces": ["packages/*"]
}`,
		"packages/core/package.json": `{"name":"core"}`,
		"packages/web/package.json":  `{"name":"web"}`,
	})

	info := detectNPMWorkspace(dir)
	if info == nil {
		t.Fatal("expected npm workspace from package.json, got nil")
	}
	if info.BuildSystem != "npm-workspace" {
		t.Errorf("BuildSystem = %q, want npm-workspace", info.BuildSystem)
	}
}

func TestDetectMultiModulePicksRightSystem(t *testing.T) {
	// Gradle project — should pick Gradle not NPM even if package.json exists
	dir := newScanDir(t, map[string]string{
		"settings.gradle":   "include 'app', 'lib'\n",
		"package.json":      `{"name":"test"}`,
		"app/build.gradle":  "// app\n",
		"lib/build.gradle":  "// lib\n",
	})

	info := DetectMultiModule(dir)
	if info == nil {
		t.Fatal("expected multi-module detection")
	}
	if info.BuildSystem != "gradle" {
		t.Errorf("BuildSystem = %q, want gradle (should prefer Gradle over NPM)", info.BuildSystem)
	}
}

// Framework detection tests

func TestDetectGrailsFramework(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"morpheus-ui/grails-app/conf/application.yml":  "server:\n  port: 8080\n",
		"morpheus-ui/grails-app/controllers/empty.txt": "",
		"morpheus-ui/grails-app/services/empty.txt":    "",
		"morpheus-ui/grails-app/views/empty.txt":       "",
		"morpheus-ui/grails-app/domain/empty.txt":      "",
		"morpheus-ui/grails-app/assets/empty.txt":      "",
		"morpheus-ui/grails-app/jobs/empty.txt":        "",
		"morpheus-ui/grails-app/migrations/empty.txt":  "",
		"gradle.properties":                            "grailsVersion=6.2.1\n",
	})

	fd := detectGrailsFramework(dir)
	if fd == nil {
		t.Fatal("expected Grails detection, got nil")
	}
	if fd.Name != "Grails" {
		t.Errorf("Name = %q, want Grails", fd.Name)
	}
	if fd.Version != "6.2.1" {
		t.Errorf("Version = %q, want 6.2.1", fd.Version)
	}
	// Check conventions detected
	hasControllers := false
	hasServices := false
	for _, c := range fd.Conventions {
		if c == "controllers" {
			hasControllers = true
		}
		if c == "services" {
			hasServices = true
		}
	}
	if !hasControllers {
		t.Error("expected controllers convention")
	}
	if !hasServices {
		t.Error("expected services convention")
	}
}

func TestDetectGrailsFrameworkNone(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"build.gradle": "// just java\n",
		"src/main/java/App.java": "class App {}",
	})

	fd := detectGrailsFramework(dir)
	if fd != nil {
		t.Error("expected nil for non-Grails project")
	}
}

func TestDetectSpringBoot(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"src/main/resources/application.properties": "server.port=8080\nspring.datasource.url=jdbc:mysql://localhost/db\n",
		"src/main/java/com/example/App.java":        "class App {}",
		"build.gradle":                              "plugins { id 'org.springframework.boot' }\n",
	})

	fd := detectSpringBoot(dir)
	if fd == nil {
		t.Fatal("expected Spring Boot detection, got nil")
	}
	if fd.Name != "Spring Boot" {
		t.Errorf("Name = %q, want Spring Boot", fd.Name)
	}
	if fd.Metadata["server-port"] != "8080" {
		t.Errorf("server-port = %q, want 8080", fd.Metadata["server-port"])
	}
}

func TestDetectSpringBootNotGrails(t *testing.T) {
	// Grails uses Spring Boot underneath, but we should detect as Grails not Spring Boot
	dir := newScanDir(t, map[string]string{
		"grails-app/conf/application.yml": "server:\n  port: 8080\n",
		"src/main/resources/application.properties": "server.port=8080\n",
		"build.gradle": "plugins { id 'org.springframework.boot' }\n",
	})

	fd := detectSpringBoot(dir)
	if fd != nil {
		t.Error("expected nil for Grails project (should not double-detect as Spring Boot)")
	}
}

func TestDetectDjango(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"manage.py": `#!/usr/bin/env python
import os
os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'myproject.settings')
from django.core.management import execute_from_command_line
execute_from_command_line()
`,
		"myproject/__init__.py": "",
		"myproject/settings.py": "# Django settings",
		"myapp/apps.py":         "# app config",
		"myapp/models.py":       "# models",
		"myapp/views.py":        "# views",
	})

	fd := detectDjango(dir)
	if fd == nil {
		t.Fatal("expected Django detection, got nil")
	}
	if fd.Name != "Django" {
		t.Errorf("Name = %q, want Django", fd.Name)
	}
	if fd.Metadata["settings-module"] != "myproject.settings" {
		t.Errorf("settings-module = %q, want myproject.settings", fd.Metadata["settings-module"])
	}
	// Should detect myapp as a Django app
	foundApp := false
	for _, c := range fd.Conventions {
		if c == "myapp" {
			foundApp = true
		}
	}
	if !foundApp {
		t.Errorf("expected myapp in conventions, got %v", fd.Conventions)
	}
}

func TestDetectRails(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"config/routes.rb":            "Rails.application.routes.draw {}\n",
		"app/models/user.rb":          "class User < ApplicationRecord; end\n",
		"app/controllers/empty.txt":   "",
		"app/views/empty.txt":         "",
		"db/migrate/001_create_users.rb": "class CreateUsers < ActiveRecord::Migration; end\n",
		"Gemfile.lock": `GEM
  remote: https://rubygems.org/
  specs:
    rails (7.1.3)
`,
	})

	fd := detectRails(dir)
	if fd == nil {
		t.Fatal("expected Rails detection, got nil")
	}
	if fd.Name != "Ruby on Rails" {
		t.Errorf("Name = %q, want Ruby on Rails", fd.Name)
	}
	if fd.Version != "7.1.3" {
		t.Errorf("Version = %q, want 7.1.3", fd.Version)
	}
}

func TestDetectLaravel(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"artisan": `#!/usr/bin/env php
<?php
require __DIR__.'/vendor/autoload.php';
$app = require_once __DIR__.'/bootstrap/app.php';
// laravel artisan
`,
		"app/Http/Controllers/empty.txt":  "",
		"app/Models/User.php":             "<?php\nclass User extends Model {}\n",
		"database/migrations/empty.txt":   "",
		"resources/views/empty.txt":       "",
	})

	fd := detectLaravel(dir)
	if fd == nil {
		t.Fatal("expected Laravel detection, got nil")
	}
	if fd.Name != "Laravel" {
		t.Errorf("Name = %q, want Laravel", fd.Name)
	}
}

func TestFindDirsNamed(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"a/grails-app/conf/x.yml":  "",
		"b/grails-app/conf/y.yml":  "",
		"a/b/c/grails-app/deep.txt": "", // depth 3 — should be excluded with maxDepth 2
	})

	found := findDirsNamed(dir, "grails-app", 2)
	if len(found) != 2 {
		t.Errorf("expected 2 grails-app dirs at depth <= 2, got %d: %v", len(found), found)
	}
}

func TestExpandGlob(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"packages/a/index.js": "",
		"packages/b/index.js": "",
		"packages/c/index.js": "",
	})

	dirs := expandGlob(dir, "packages/*")
	if len(dirs) != 3 {
		t.Errorf("expected 3 dirs from glob, got %d: %v", len(dirs), dirs)
	}
}

func TestDirExists(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"subdir/file.txt": "",
	})

	if !dirExists(dir, "subdir") {
		t.Error("dirExists should return true for existing directory")
	}
	if dirExists(dir, "nonexistent") {
		t.Error("dirExists should return false for missing directory")
	}
}

func TestEnrichGradleSubprojectGroovy(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"mymod/src/main/groovy/Foo.groovy": "class Foo {}",
	})

	sp := &Subproject{Name: "mymod", Dir: "mymod", Kind: "gradle-module"}
	enrichGradleSubproject(dir, sp)

	if sp.Language != "Groovy" {
		t.Errorf("Language = %q, want Groovy", sp.Language)
	}
}

func TestDetectGrailsVersion(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name:  "from gradle.properties",
			files: map[string]string{"gradle.properties": "grailsVersion=6.2.1\norg.gradle.daemon=true\n"},
			want:  "6.2.1",
		},
		{
			name:  "from build.gradle",
			files: map[string]string{"build.gradle": `ext { grailsVersion = "5.3.0" }`},
			want:  "5.3.0",
		},
		{
			name:  "not found",
			files: map[string]string{"build.gradle": "// no version"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				full := filepath.Join(dir, name)
				os.MkdirAll(filepath.Dir(full), 0o755)
				os.WriteFile(full, []byte(content), 0o644)
			}
			got := detectGrailsVersion(dir)
			if got != tt.want {
				t.Errorf("detectGrailsVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
