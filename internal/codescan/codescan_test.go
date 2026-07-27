package codescan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

type failingParser struct{}

func (failingParser) Languages() []string { return []string{"go"} }
func (failingParser) ParseFile(string, []byte) (*FileInfo, error) {
	return nil, errors.New("injected parse failure")
}

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
		"useCart": true,
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
	result, err := scanner.Scan(nil, nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.Packages) != 2 {
		t.Errorf("Packages = %d, want 2", len(result.Packages))
	}
	if len(result.Checksums) != 2 {
		t.Errorf("Checksums = %d, want 2", len(result.Checksums))
	}

	// Test incremental scan — no changes; unchanged files are carried forward
	// from the cache rather than re-parsed.
	result2, err := scanner.Scan(result.Checksums, BuildScanCache(result))
	if err != nil {
		t.Fatalf("Incremental scan failed: %v", err)
	}
	if len(result2.Checksums) != 2 {
		t.Errorf("Incremental checksums = %d, want 2", len(result2.Checksums))
	}
}

func TestScanContextAccountsForChangesAndEndpointOnlySources(t *testing.T) {
	dir := t.TempDir()
	goPath := filepath.Join(dir, "main.go")
	protoPath := filepath.Join(dir, "service.proto")
	if err := os.WriteFile(goPath, []byte("package main\nfunc Main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(protoPath, []byte("service Greeter { rpc Hello (Req) returns (Resp); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner(config.DefaultCodeScanConfig(), dir)
	first, err := scanner.ScanContext(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if !first.Complete {
		t.Fatalf("first scan incomplete: %+v", first.Diagnostics)
	}
	if first.Stats.FilesInventoried != 2 || first.Stats.Reparsed != 2 || first.Stats.Added != 2 {
		t.Fatalf("first stats = %+v, want two inventoried/reparsed/added", first.Stats)
	}
	if len(first.Checksums) != 2 {
		t.Fatalf("checksums = %v, endpoint-only source was not inventoried", first.Checksums)
	}

	second, err := scanner.ScanContext(context.Background(), first.Checksums, BuildScanCache(first))
	if err != nil {
		t.Fatalf("unchanged scan: %v", err)
	}
	if second.Stats.Reparsed != 0 || second.Stats.Unchanged != 2 {
		t.Fatalf("unchanged stats = %+v, want zero reparsed and two unchanged", second.Stats)
	}

	if err := os.WriteFile(goPath, []byte("package main\nfunc Main() {}\nfunc Added() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(protoPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "schema.graphql"), []byte("type Query { hero: String }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := scanner.ScanContext(context.Background(), second.Checksums, BuildScanCache(second))
	if err != nil {
		t.Fatalf("changed scan: %v", err)
	}
	if third.Stats.Added != 1 || third.Stats.Changed != 1 || third.Stats.Deleted != 1 || third.Stats.Reparsed != 2 {
		t.Fatalf("changed stats = %+v, want add/change/delete=1 and reparsed=2", third.Stats)
	}
}

func TestScanContextMarksParseFailureIncomplete(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scanner := NewScanner(config.DefaultCodeScanConfig(), dir)
	scanner.parsers["go"] = failingParser{}
	result, err := scanner.ScanContext(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("ScanContext: %v", err)
	}
	if result.Complete {
		t.Fatal("parse failure must make the source snapshot incomplete")
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Phase != "parse" {
		t.Fatalf("diagnostics = %+v, want one parse failure", result.Diagnostics)
	}
	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")
	if err := GenerateKnowledgeContext(context.Background(), result, codeDir); err == nil {
		t.Fatal("incomplete result must not generate authoritative knowledge")
	}
	if err := CommitScanState(result, codeDir, "heuristic"); err == nil {
		t.Fatal("incomplete result must not advance scan state")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scanner.ScanContext(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled ScanContext error = %v, want context.Canceled", err)
	}
}

func TestScanStateRejectsSplitGenerationAndParserMismatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc Main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := NewScanner(config.DefaultCodeScanConfig(), dir).Scan(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")
	if err := CommitScanState(result, codeDir, "heuristic"); err != nil {
		t.Fatalf("CommitScanState: %v", err)
	}
	checksums, cache, usable, err := LoadScanState(codeDir, "heuristic")
	if err != nil || !usable || cache == nil || len(checksums) != 1 {
		t.Fatalf("LoadScanState = (%v, %+v, %v, %v), want usable pair", checksums, cache, usable, err)
	}
	if cache.Generation == "" || cache.ChecksumsHash == "" {
		t.Fatalf("cache manifest missing generation/hash: %+v", cache)
	}

	if err := SaveChecksums(codeDir, map[string]string{"main.go": "split"}); err != nil {
		t.Fatal(err)
	}
	if _, _, usable, err := LoadScanState(codeDir, "heuristic"); err != nil || usable {
		t.Fatalf("split pair usable=%v err=%v, want rejected without load error", usable, err)
	}

	if err := CommitScanState(result, codeDir, "heuristic"); err != nil {
		t.Fatal(err)
	}
	if _, _, usable, err := LoadScanState(codeDir, "treesitter"); err != nil || usable {
		t.Fatalf("parser-mismatched pair usable=%v err=%v, want rejected", usable, err)
	}
}

func TestGenerateKnowledgeContextCommitsStateSeparately(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc Main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := NewScanner(config.DefaultCodeScanConfig(), dir).Scan(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")
	if err := GenerateKnowledgeContext(context.Background(), result, codeDir); err != nil {
		t.Fatalf("GenerateKnowledgeContext: %v", err)
	}
	for _, name := range []string{".checksums.json", ".scan-cache.json"} {
		if _, err := os.Stat(filepath.Join(codeDir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s exists before state commit (err=%v)", name, err)
		}
	}
	if err := CommitScanState(result, codeDir, "heuristic"); err != nil {
		t.Fatalf("CommitScanState: %v", err)
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
	result, err := scanner.Scan(nil, nil)
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

// TestGenerateKnowledgeIncrementalPreservesUnchangedPackages guards against the
// data-loss bug where an incremental scan (which rebuilds only changed packages)
// pruned every unchanged package's directory from .hero/knowledge/code/. The
// prune keep-set must be derived from the complete current file set
// (result.Checksums), not the changed-only result.Packages.
func TestGenerateKnowledgeIncrementalPreservesUnchangedPackages(t *testing.T) {
	dir := t.TempDir()

	aDir := filepath.Join(dir, "internal", "a")
	os.MkdirAll(aDir, 0o755)
	aFile := filepath.Join(aDir, "a.go")
	os.WriteFile(aFile, []byte(`package a

// Alpha does alpha things.
type Alpha struct{}
`), 0o644)

	bDir := filepath.Join(dir, "internal", "b")
	os.MkdirAll(bDir, 0o755)
	bFile := filepath.Join(bDir, "b.go")
	os.WriteFile(bFile, []byte(`package b

// Beta does beta things.
type Beta struct{}
`), 0o644)

	cfg := config.DefaultCodeScanConfig()
	scanner := NewScanner(cfg, dir)
	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")

	// Full scan writes both packages.
	result, err := scanner.Scan(nil, nil)
	if err != nil {
		t.Fatalf("full scan failed: %v", err)
	}
	if err := GenerateKnowledge(result, codeDir); err != nil {
		t.Fatalf("GenerateKnowledge (full) failed: %v", err)
	}
	aSpec := filepath.Join(codeDir, "internal-a", "spec.md")
	bSpec := filepath.Join(codeDir, "internal-b", "spec.md")
	if _, err := os.Stat(aSpec); err != nil {
		t.Fatalf("package A spec missing after full scan: %v", err)
	}
	if _, err := os.Stat(bSpec); err != nil {
		t.Fatalf("package B spec missing after full scan: %v", err)
	}

	// Change ONLY package A, then re-scan incrementally with the prior checksums.
	os.WriteFile(aFile, []byte(`package a

// Alpha does alpha things.
type Alpha struct{}

// Gamma is new.
func Gamma() {}
`), 0o644)

	result2, err := scanner.Scan(result.Checksums, BuildScanCache(result))
	if err != nil {
		t.Fatalf("incremental scan failed: %v", err)
	}
	if err := GenerateKnowledge(result2, codeDir); err != nil {
		t.Fatalf("GenerateKnowledge (incremental) failed: %v", err)
	}

	// Core regression: package B was unchanged but its spec.md must survive.
	if _, err := os.Stat(bSpec); err != nil {
		t.Errorf("package B spec deleted by incremental scan (regression): %v", err)
	}
	// Package A was rewritten and must still exist.
	if _, err := os.Stat(aSpec); err != nil {
		t.Errorf("package A spec missing after incremental scan: %v", err)
	}

	// Genuinely-deleted package: remove B's file, re-scan incrementally, and
	// confirm B's directory IS pruned (deletions must still take effect).
	os.Remove(bFile)
	result3, err := scanner.Scan(result2.Checksums, BuildScanCache(result2))
	if err != nil {
		t.Fatalf("incremental scan after delete failed: %v", err)
	}
	if err := GenerateKnowledge(result3, codeDir); err != nil {
		t.Fatalf("GenerateKnowledge (after delete) failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(codeDir, "internal-b")); !os.IsNotExist(err) {
		t.Errorf("package B dir should be pruned after its file was deleted, stat err = %v", err)
	}
	if _, err := os.Stat(aSpec); err != nil {
		t.Errorf("package A spec missing after delete-scan: %v", err)
	}
}

// --- Incremental-scan-complete-result tests ---
//
// These lock in that an incremental scan produces a Result equal by content to
// a full scan of the same tree (packages, config vars, endpoints), by carrying
// unchanged files forward from the scan cache. Comparisons are order-insensitive
// because full and incremental scans append in different file-walk order.

// canonPackages renders packages as a sorted, order-insensitive canonical form:
// per package, its path, language, line/file counts, sorted files, and sorted
// symbols (name|kind|file|line). This is what the equivalence assertions compare.
func canonPackages(pkgs []Package) []string {
	out := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		files := append([]string(nil), p.Files...)
		sort.Strings(files)
		syms := make([]string, 0, len(p.Symbols))
		for _, s := range p.Symbols {
			syms = append(syms, fmt.Sprintf("%s|%s|%s|%d", s.Name, s.Kind, s.File, s.Line))
		}
		sort.Strings(syms)
		out = append(out, fmt.Sprintf("path=%s lang=%s lines=%d files=%d [%s] {%s}",
			p.Path, p.Language, p.LineCount, p.FileCount,
			strings.Join(files, ","), strings.Join(syms, ",")))
	}
	sort.Strings(out)
	return out
}

func canonConfigVars(cvs []ConfigVar) []string {
	out := make([]string, 0, len(cvs))
	for _, c := range cvs {
		out = append(out, fmt.Sprintf("%s|%s|%s|%d|%v", c.Name, c.Source, c.File, c.Line, c.Required))
	}
	sort.Strings(out)
	return out
}

func canonEndpoints(eps []Endpoint) []string {
	out := make([]string, 0, len(eps))
	for _, e := range eps {
		out = append(out, fmt.Sprintf("%s|%s|%s|%s|%d|%s", e.Method, e.Path, e.Handler, e.File, e.Line, e.Protocol))
	}
	sort.Strings(out)
	return out
}

func assertStringSlicesEqual(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: length mismatch got=%d want=%d\n got=%v\nwant=%v", label, len(got), len(want), got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: element %d mismatch\n got=%q\nwant=%q", label, i, got[i], want[i])
		}
	}
}

// writeEquivFixture builds a project with three package shapes:
//   - package A (internal/a): two files; a1.go will change, a2.go stays put.
//     a1.go reads an env var (ConfigVar in the changed file); a2.go registers a
//     route (Endpoint in the UNCHANGED file — the carry-forward path under test).
//   - package B (internal/b): untouched, single file with its own symbol.
//   - package C (internal/c): single file that will change.
func writeEquivFixture(t *testing.T) (dir, a1, cFile string) {
	t.Helper()
	dir = t.TempDir()

	aDir := filepath.Join(dir, "internal", "a")
	if err := os.MkdirAll(aDir, 0o755); err != nil {
		t.Fatal(err)
	}
	a1 = filepath.Join(aDir, "a1.go")
	os.WriteFile(a1, []byte(`package a

import "os"

// DBURL reads the database url.
func DBURL() string { return os.Getenv("DATABASE_URL") }
`), 0o644)
	os.WriteFile(filepath.Join(aDir, "a2.go"), []byte(`package a

import "net/http"

// RegisterHealth wires the health route.
func RegisterHealth() {
	http.HandleFunc("/health", nil)
}
`), 0o644)

	bDir := filepath.Join(dir, "internal", "b")
	if err := os.MkdirAll(bDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(bDir, "b.go"), []byte(`package b

// Beta does beta things.
type Beta struct{}
`), 0o644)

	cDir := filepath.Join(dir, "internal", "c")
	if err := os.MkdirAll(cDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cFile = filepath.Join(cDir, "c.go")
	os.WriteFile(cFile, []byte(`package c

// Charlie does charlie things.
func Charlie() {}
`), 0o644)

	return dir, a1, cFile
}

// TestIncrementalScanEqualsFullScan is the core equivalence test: after mutating
// one file in a partially-changed package plus a single-file package, an
// incremental scan produces packages/configvars/endpoints equal by content to a
// full re-scan, and the partially-changed package's on-disk spec.md lists BOTH
// files' symbols.
func TestIncrementalScanEqualsFullScan(t *testing.T) {
	dir, a1, cFile := writeEquivFixture(t)
	cfg := config.DefaultCodeScanConfig()
	scanner := NewScanner(cfg, dir)
	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")

	r1, err := scanner.Scan(nil, nil)
	if err != nil {
		t.Fatalf("full scan failed: %v", err)
	}

	// Mutate A's a1.go (adds a symbol, keeps the env var) and C's file.
	os.WriteFile(a1, []byte(`package a

import "os"

// DBURL reads the database url.
func DBURL() string { return os.Getenv("DATABASE_URL") }

// CacheURL is new in a1.
func CacheURL() string { return os.Getenv("CACHE_URL") }
`), 0o644)
	os.WriteFile(cFile, []byte(`package c

// Charlie does charlie things.
func Charlie() {}

// Delta is new.
func Delta() {}
`), 0o644)

	// Incremental scan carrying unchanged files forward from r1's cache.
	r2, err := scanner.Scan(r1.Checksums, BuildScanCache(r1))
	if err != nil {
		t.Fatalf("incremental scan failed: %v", err)
	}

	// Full re-scan of the mutated tree with a fresh scanner (no cache).
	fresh := NewScanner(cfg, dir)
	rFull, err := fresh.Scan(nil, nil)
	if err != nil {
		t.Fatalf("full re-scan failed: %v", err)
	}

	assertStringSlicesEqual(t, "packages", canonPackages(r2.Packages), canonPackages(rFull.Packages))
	assertStringSlicesEqual(t, "config_vars", canonConfigVars(r2.ConfigVars), canonConfigVars(rFull.ConfigVars))
	assertStringSlicesEqual(t, "endpoints", canonEndpoints(r2.Endpoints), canonEndpoints(rFull.Endpoints))

	// The carry-forward path must preserve the endpoint from the UNCHANGED file.
	if len(canonEndpoints(r2.Endpoints)) == 0 {
		t.Fatal("expected at least one endpoint carried forward from a2.go")
	}

	// Deepest defect: package A's spec.md must list BOTH files' symbols after an
	// incremental scan, not just the changed file's.
	if err := GenerateKnowledge(r2, codeDir); err != nil {
		t.Fatalf("GenerateKnowledge failed: %v", err)
	}
	aSpec, err := os.ReadFile(filepath.Join(codeDir, "internal-a", "spec.md"))
	if err != nil {
		t.Fatalf("reading package A spec: %v", err)
	}
	s := string(aSpec)
	for _, want := range []string{"DBURL", "CacheURL", "RegisterHealth"} {
		if !contains(s, want) {
			t.Errorf("package A spec.md missing symbol %q (partial-package corruption)", want)
		}
	}
}

// TestIncrementalScanDeletedFileDropsAndPrunes confirms a file deleted between
// scans drops its package (when emptied) and its extracted symbols/endpoints.
func TestIncrementalScanDeletedFileDropsAndPrunes(t *testing.T) {
	dir, _, _ := writeEquivFixture(t)
	cfg := config.DefaultCodeScanConfig()
	scanner := NewScanner(cfg, dir)
	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")

	r1, err := scanner.Scan(nil, nil)
	if err != nil {
		t.Fatalf("full scan failed: %v", err)
	}
	if err := GenerateKnowledge(r1, codeDir); err != nil {
		t.Fatalf("GenerateKnowledge (full) failed: %v", err)
	}

	// Delete package B's only file.
	os.Remove(filepath.Join(dir, "internal", "b", "b.go"))

	r2, err := scanner.Scan(r1.Checksums, BuildScanCache(r1))
	if err != nil {
		t.Fatalf("incremental scan failed: %v", err)
	}
	if err := GenerateKnowledge(r2, codeDir); err != nil {
		t.Fatalf("GenerateKnowledge (incremental) failed: %v", err)
	}

	for _, p := range r2.Packages {
		if p.Path == filepath.Join("internal", "b") {
			t.Errorf("package B should be absent from incremental result, found %+v", p)
		}
	}
	if _, err := os.Stat(filepath.Join(codeDir, "internal-b")); !os.IsNotExist(err) {
		t.Errorf("package B dir should be pruned, stat err = %v", err)
	}
	// The deleted package's endpoint/configvars must be gone; the surviving
	// packages' endpoints (from a2.go) must remain.
	assertStringSlicesEqual(t, "packages match fresh scan",
		canonPackages(r2.Packages), canonPackages(mustFullScan(t, cfg, dir).Packages))
}

// TestIncrementalScanMissingCacheFallback confirms that a nil cache and a
// corrupted .scan-cache.json both degrade to a complete (full-equivalent) result.
func TestIncrementalScanMissingCacheFallback(t *testing.T) {
	dir, a1, _ := writeEquivFixture(t)
	cfg := config.DefaultCodeScanConfig()
	scanner := NewScanner(cfg, dir)
	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")

	r1, err := scanner.Scan(nil, nil)
	if err != nil {
		t.Fatalf("full scan failed: %v", err)
	}

	// Change a1.go so an incremental scan must re-parse it while a2.go would
	// normally carry forward — but here there is no usable cache.
	os.WriteFile(a1, []byte(`package a

import "os"

// DBURL reads the database url.
func DBURL() string { return os.Getenv("DATABASE_URL") }

// Extra is new.
func Extra() {}
`), 0o644)

	want := canonPackages(mustFullScan(t, cfg, dir).Packages)

	// (a) nil cache.
	rNil, err := scanner.Scan(r1.Checksums, nil)
	if err != nil {
		t.Fatalf("incremental scan (nil cache) failed: %v", err)
	}
	assertStringSlicesEqual(t, "nil-cache packages", canonPackages(rNil.Packages), want)

	// (b) corrupted .scan-cache.json → LoadScanCache returns a non-nil error and
	// a nil cache; the caller proceeds with nil.
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeDir, ".scan-cache.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	corrupt, loadErr := LoadScanCache(codeDir)
	if loadErr == nil {
		t.Error("expected LoadScanCache to return an error for corrupted cache")
	}
	if corrupt != nil {
		t.Errorf("expected nil cache for corrupted file, got %+v", corrupt)
	}
	rCorrupt, err := scanner.Scan(r1.Checksums, corrupt)
	if err != nil {
		t.Fatalf("incremental scan (corrupt cache) failed: %v", err)
	}
	assertStringSlicesEqual(t, "corrupt-cache packages", canonPackages(rCorrupt.Packages), want)
}

// TestFullScanWritesCompleteCache confirms a full scan + GenerateKnowledge writes
// a versioned .scan-cache.json that round-trips every current file.
func TestFullScanWritesCompleteCache(t *testing.T) {
	dir, _, _ := writeEquivFixture(t)
	cfg := config.DefaultCodeScanConfig()
	scanner := NewScanner(cfg, dir)
	codeDir := filepath.Join(dir, ".hero", "knowledge", "code")

	r1, err := scanner.Scan(nil, nil)
	if err != nil {
		t.Fatalf("full scan failed: %v", err)
	}
	if err := GenerateKnowledge(r1, codeDir); err != nil {
		t.Fatalf("GenerateKnowledge failed: %v", err)
	}

	cachePath := filepath.Join(codeDir, ".scan-cache.json")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf(".scan-cache.json not written: %v", err)
	}

	cache, err := LoadScanCache(codeDir)
	if err != nil {
		t.Fatalf("LoadScanCache failed: %v", err)
	}
	if cache == nil {
		t.Fatal("expected a non-nil cache after a full scan")
	}
	if cache.Version != scanCacheVersion {
		t.Errorf("cache version = %d, want %d", cache.Version, scanCacheVersion)
	}
	if len(r1.Files) == 0 {
		t.Fatal("expected r1.Files to be non-empty")
	}
	for _, fi := range r1.Files {
		if _, ok := cache.Files[fi.Path]; !ok {
			t.Errorf("cache missing entry for file %q", fi.Path)
		}
	}
}

func mustFullScan(t *testing.T, cfg *config.CodeScanConfig, dir string) *Result {
	t.Helper()
	r, err := NewScanner(cfg, dir).Scan(nil, nil)
	if err != nil {
		t.Fatalf("full scan failed: %v", err)
	}
	return r
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
