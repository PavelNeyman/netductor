#!/usr/bin/env bash
# Phase 3: ensure Go contract required assets stay aligned (unit tests).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
go test ./internal/update/ -count=1 -run 'RequiredAssets|ValidateSUMS|UserFacing|AssetName'
echo "check-update-asset-matrix: OK"
