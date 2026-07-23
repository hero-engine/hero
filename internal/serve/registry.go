package serve

import "github.com/hero-engine/hero/internal/projectregistry"

type ProjectEntry = projectregistry.ProjectEntry
type Registry = projectregistry.Registry

func registryDir() (string, error)                    { return projectregistry.Dir() }
func registryPath() (string, error)                   { return projectregistry.Path() }
func LoadRegistry() (*Registry, error)                { return projectregistry.Load() }
func LoadRegistryFrom(path string) (*Registry, error) { return projectregistry.LoadFrom(path) }
