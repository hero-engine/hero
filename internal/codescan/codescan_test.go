package codescan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestGoParser(t *testing.T) {
	src := []byte(`package example

import (
	"context"
	"fmt"
)

// UserService handles user operations.
type UserService struct {
	db *DB
}

// User represents a user entity.
type User struct {
	ID   string
	Name string
}

// Storer is the storage interface.
type Storer interface {
	Get(id string) (*User, error)
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
	return nil, fmt.Errorf("not implemented")
}

// NewUserService creates a new service.
func NewUserService(db *DB) *UserService {
	return &UserService{db: db}
}

const DefaultPageSize = 20

var globalCache map[string]*User
`)

	p := NewGoParser()
	fi, err := p.ParseFile("internal/user/service.go", src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if fi.Language != "go" {
		t.Errorf("Language = %q, want %q", fi.Language, "go")
	}
	if fi.Package != "example" {
		t.Errorf("Package = %q, want %q", fi.Package, "example")
	}
	if len(fi.Imports) != 2 {
		t.Errorf("Imports = %d, want 2", len(fi.Imports))
	}

	// Check symbols
	wantSymbols := map[string]SymbolKind{
		"UserService":     SymStruct,
		"User":            SymStruct,
		"Storer":          SymInterface,
		"GetUser":         SymMethod,
		"NewUserService":  SymFunc,
		"DefaultPageSize": SymConst,
		"globalCache":     SymVar,
	}

	for _, sym := range fi.Symbols {
		if expected, ok := wantSymbols[sym.Name]; ok {
			if sym.Kind != expected {
				t.Errorf("Symbol %q: kind = %q, want %q", sym.Name, sym.Kind, expected)
			}
			delete(wantSymbols, sym.Name)
		}
	}
	for name := range wantSymbols {
		t.Errorf("Missing symbol: %s", name)
	}

	// Check exported
	for _, sym := range fi.Symbols {
		switch sym.Name {
		case "UserService", "User", "Storer", "GetUser", "NewUserService", "DefaultPageSize":
			if !sym.Exported {
				t.Errorf("Symbol %q should be exported", sym.Name)
			}
		case "globalCache":
			if sym.Exported {
				t.Errorf("Symbol %q should not be exported", sym.Name)
			}
		}
	}

	// Check doc extraction
	for _, sym := range fi.Symbols {
		if sym.Name == "UserService" && sym.Doc == "" {
			t.Error("UserService should have doc comment")
		}
		if sym.Name == "GetUser" && sym.Doc == "" {
			t.Error("GetUser should have doc comment")
		}
	}

	// Check method receiver
	for _, sym := range fi.Symbols {
		if sym.Name == "GetUser" {
			if sym.Receiver == "" {
				t.Error("GetUser should have a receiver")
			}
		}
	}
}

func TestJSTSParser(t *testing.T) {
	src := []byte(`import { useState } from 'react';
import axios from 'axios';

export interface UserProps {
  name: string;
}

export type UserID = string;

export enum Role {
  Admin,
  User,
}

export class UserManager {
  constructor() {}
}

export function getUser(id: string) {
  return null;
}

export const createUser = async (data: any) => {
  return null;
};

export const MAX_USERS = 100;

function internalHelper() {}
`)

	p := NewJSTSParser()
	fi, err := p.ParseFile("src/user.ts", src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if fi.Language != "typescript" {
		t.Errorf("Language = %q, want %q", fi.Language, "typescript")
	}
	if len(fi.Imports) != 2 {
		t.Errorf("Imports = %d, want 2", len(fi.Imports))
	}

	wantExported := map[string]bool{
		"UserProps":   true,
		"UserID":      true,
		"Role":        true,
		"UserManager": true,
		"getUser":     true,
		"createUser":  true,
		"MAX_USERS":   true,
	}

	for _, sym := range fi.Symbols {
		if wantExported[sym.Name] {
			if !sym.Exported {
				t.Errorf("Symbol %q should be exported", sym.Name)
			}
			delete(wantExported, sym.Name)
		}
		if sym.Name == "internalHelper" && sym.Exported {
			t.Error("internalHelper should not be exported")
		}
	}
	for name := range wantExported {
		t.Errorf("Missing exported symbol: %s", name)
	}
}

func TestJSTSParser_React(t *testing.T) {
	src := []byte(`import React from 'react';
import { useEffect } from 'react';

// export default function component
export default function App() {
  return <div />;
}

// React hook
export function useAuth() {
  return { user: null };
}

// Type-annotated arrow component
export const Button: React.FC<ButtonProps> = ({ children }) => {
  return <button>{children}</button>;
};

// Arrow hook
export const useCart = () => {
  return [];
};

// Non-exported component, exported at end
const Footer = () => {
  return <footer />;
};

export default Footer;
`)

	p := NewJSTSParser()
	fi, err := p.ParseFile("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	wantExported := map[string]bool{
		"App":     true,
		"useAuth": true,
		"Button":  true,
		"useCart":  true,
		"Footer":  true,
	}

	for _, sym := range fi.Symbols {
		if wantExported[sym.Name] {
			if !sym.Exported {
				t.Errorf("Symbol %q should be exported", sym.Name)
			}
			delete(wantExported, sym.Name)
		}
	}
	for name := range wantExported {
		t.Errorf("Missing exported symbol: %s", name)
	}

	// Button should be detected as func (not const)
	for _, sym := range fi.Symbols {
		if sym.Name == "Button" && sym.Kind != SymFunc {
			t.Errorf("Button kind = %q, want func (type-annotated arrow component)", sym.Kind)
		}
	}
}

func TestJSTSParser_ReExports(t *testing.T) {
	src := []byte(`export { default as Button } from './Button';
export { Input, Label as FormLabel } from './forms';
export * from './utils';
export { foo, bar, baz };
`)

	p := NewJSTSParser()
	fi, err := p.ParseFile("src/index.ts", src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should capture re-export imports
	if len(fi.Imports) < 2 {
		t.Errorf("Imports = %d, want at least 2 (./Button, ./forms or ./utils)", len(fi.Imports))
	}

	// Should capture re-exported names
	wantExported := map[string]bool{
		"Button":    true,
		"Input":     true,
		"FormLabel": true,
		"foo":       true,
		"bar":       true,
		"baz":       true,
	}

	for _, sym := range fi.Symbols {
		if wantExported[sym.Name] {
			if !sym.Exported {
				t.Errorf("Symbol %q should be exported", sym.Name)
			}
			delete(wantExported, sym.Name)
		}
	}
	for name := range wantExported {
		t.Errorf("Missing exported symbol: %s", name)
	}
}

func TestJSTSParser_CommonJS(t *testing.T) {
	src := []byte(`const express = require('express');

function createApp() {
  return express();
}

function setupRoutes(app) {
  // ...
}

module.exports = { createApp, setupRoutes };
`)

	p := NewJSTSParser()
	fi, err := p.ParseFile("src/app.js", src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if fi.Language != "javascript" {
		t.Errorf("Language = %q, want javascript", fi.Language)
	}

	// require should be captured as import
	foundImport := false
	for _, imp := range fi.Imports {
		if imp == "express" {
			foundImport = true
		}
	}
	if !foundImport {
		t.Error("Missing import: express")
	}

	// module.exports should mark functions as exported
	for _, sym := range fi.Symbols {
		if sym.Name == "createApp" || sym.Name == "setupRoutes" {
			if !sym.Exported {
				t.Errorf("Symbol %q should be exported (via module.exports)", sym.Name)
			}
		}
	}
}

func TestJSTSParser_DestructuredExport(t *testing.T) {
	src := []byte(`export const { Provider, Consumer } = React.createContext();
`)

	p := NewJSTSParser()
	fi, err := p.ParseFile("src/context.tsx", src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	wantExported := map[string]bool{
		"Provider": true,
		"Consumer": true,
	}

	for _, sym := range fi.Symbols {
		if wantExported[sym.Name] {
			if !sym.Exported {
				t.Errorf("Symbol %q should be exported", sym.Name)
			}
			delete(wantExported, sym.Name)
		}
	}
	for name := range wantExported {
		t.Errorf("Missing exported symbol: %s", name)
	}
}

func TestPythonParser(t *testing.T) {
	src := []byte(`import os
from pathlib import Path

MAX_RETRIES = 3

class UserService:
    def __init__(self):
        pass

    def get_user(self, user_id):
        pass

def create_user(data):
    pass

def _internal_helper():
    pass
`)

	p := NewPythonParser()
	fi, err := p.ParseFile("services/user.py", src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if fi.Language != "python" {
		t.Errorf("Language = %q, want %q", fi.Language, "python")
	}

	var foundClass, foundFunc, foundConst bool
	for _, sym := range fi.Symbols {
		switch sym.Name {
		case "UserService":
			foundClass = true
			if sym.Kind != SymClass {
				t.Errorf("UserService kind = %q, want class", sym.Kind)
			}
		case "create_user":
			foundFunc = true
			if sym.Kind != SymFunc {
				t.Errorf("create_user kind = %q, want func", sym.Kind)
			}
		case "MAX_RETRIES":
			foundConst = true
		case "_internal_helper":
			if sym.Exported {
				t.Error("_internal_helper should not be exported")
			}
		}
	}
	if !foundClass {
		t.Error("Missing UserService class")
	}
	if !foundFunc {
		t.Error("Missing create_user function")
	}
	if !foundConst {
		t.Error("Missing MAX_RETRIES constant")
	}
}

func TestRustParser(t *testing.T) {
	src := []byte(`use std::collections::HashMap;

pub struct Config {
    name: String,
}

pub enum Mode {
    Fast,
    Safe,
}

pub trait Handler {
    fn handle(&self);
}

pub fn process(data: &[u8]) -> Result<(), Error> {
    Ok(())
}

fn internal_fn() {}

pub const MAX_SIZE: usize = 1024;
`)

	p := NewRustParser()
	fi, err := p.ParseFile("src/lib.rs", src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	wantExported := map[string]SymbolKind{
		"Config":   SymStruct,
		"Mode":     SymEnum,
		"Handler":  SymTrait,
		"process":  SymFunc,
		"MAX_SIZE": SymConst,
	}

	for _, sym := range fi.Symbols {
		if expected, ok := wantExported[sym.Name]; ok {
			if sym.Kind != expected {
				t.Errorf("%s kind = %q, want %q", sym.Name, sym.Kind, expected)
			}
			if !sym.Exported {
				t.Errorf("%s should be exported", sym.Name)
			}
			delete(wantExported, sym.Name)
		}
		if sym.Name == "internal_fn" && sym.Exported {
			t.Error("internal_fn should not be exported")
		}
	}
	for name := range wantExported {
		t.Errorf("Missing symbol: %s", name)
	}
}

func TestScanner(t *testing.T) {
	// Create a temp project with Go and TS files
	dir := t.TempDir()

	// Go files
	goDir := filepath.Join(dir, "pkg", "user")
	os.MkdirAll(goDir, 0o755)
	os.WriteFile(filepath.Join(goDir, "user.go"), []byte(`package user

// User is a user.
type User struct {
	ID string
}

// GetByID finds a user.
func GetByID(id string) *User { return nil }
`), 0o644)

	// TS files
	tsDir := filepath.Join(dir, "src")
	os.MkdirAll(tsDir, 0o755)
	os.WriteFile(filepath.Join(tsDir, "api.ts"), []byte(`import { User } from './models';

export function fetchUser(id: string): Promise<User> {
  return fetch('/api/users/' + id).then(r => r.json());
}

export interface ApiConfig {
  baseUrl: string;
}
`), 0o644)

	cfg := config.DefaultCodeScanConfig()
	scanner := NewScanner(cfg, dir)
	result, err := scanner.Scan(nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.Packages) != 2 {
		t.Errorf("Packages = %d, want 2", len(result.Packages))
	}
	if len(result.Checksums) != 2 {
		t.Errorf("Checksums = %d, want 2", len(result.Checksums))
	}

	// Test incremental scan — no changes
	result2, err := scanner.Scan(result.Checksums)
	if err != nil {
		t.Fatalf("Incremental scan failed: %v", err)
	}
	// Files are skipped but checksums are still computed
	if len(result2.Checksums) != 2 {
		t.Errorf("Incremental checksums = %d, want 2", len(result2.Checksums))
	}
}

func TestGenerateKnowledge(t *testing.T) {
	dir := t.TempDir()
	goDir := filepath.Join(dir, "internal", "svc")
	os.MkdirAll(goDir, 0o755)
	os.WriteFile(filepath.Join(goDir, "svc.go"), []byte(`package svc

// Service does things.
type Service struct{}

// Run starts the service.
func (s *Service) Run() error { return nil }
`), 0o644)

	cfg := config.DefaultCodeScanConfig()
	scanner := NewScanner(cfg, dir)
	result, err := scanner.Scan(nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")
	if err := GenerateKnowledge(result, codeDir); err != nil {
		t.Fatalf("GenerateKnowledge failed: %v", err)
	}

	// Check index file exists
	indexPath := filepath.Join(codeDir, "index", "spec.md")
	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("Index file not found: %v", err)
	}

	// Check package file exists
	pkgPath := filepath.Join(codeDir, "internal-svc", "spec.md")
	if _, err := os.Stat(pkgPath); err != nil {
		t.Errorf("Package file not found: %v", err)
	}

	// Check checksums file exists
	csPath := filepath.Join(codeDir, ".checksums.json")
	if _, err := os.Stat(csPath); err != nil {
		t.Errorf("Checksums file not found: %v", err)
	}

	// Read and verify content
	content, _ := os.ReadFile(pkgPath)
	s := string(content)
	if !contains(s, "Service") {
		t.Error("Package spec should contain Service symbol")
	}
	if !contains(s, "Run") {
		t.Error("Package spec should contain Run method")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findIndex(s, substr) >= 0
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
