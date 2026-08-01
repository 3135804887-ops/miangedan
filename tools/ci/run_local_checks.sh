#!/usr/bin/env bash
# 本地一键检查：与 .github/workflows/ci.yml 阶段 1~4、6 同源。
# 阶段 5（gitleaks、依赖审查、SBOM）依赖 CI 平台动作，本地等价物见本脚本末尾。
# 用法：bash tools/ci/run_local_checks.sh
set -euo pipefail
cd "$(dirname "$0")/../.."

echo "== 阶段1 规范校验 =="
python tools/validate_docs.py --suites required,fences,placeholders,coverage,consistency,semantics,regions

echo "== 阶段2 静态检查 =="
test -z "$(gofmt -l services/)"
# go.work 根目录不支持 ./... 通配，Go 命令逐模块执行（与 ci.yml 一致）
for m in services/*/; do (cd "$m" && go vet ./...); done
# 本机装有 golangci-lint 时按 CI 矩阵同源逐模块执行；未安装则提示跳过（CI 门禁不受影响）
if command -v golangci-lint >/dev/null 2>&1; then
  for m in services/*/; do (cd "$m" && golangci-lint run --config ../../.golangci.yml ./...); done
else
  echo "提示：未安装 golangci-lint，跳过本地 golangci-lint（CI 阶段2 golangci 矩阵仍会强制执行）"
fi
for d in ai/services/*/; do (cd "$d" && ruff check . && ruff format --check .); done
for d in ai/services/*/; do (cd "$d" && mypy src tests); done

echo "== 阶段3 单元测试 =="
for m in services/*/; do (cd "$m" && go test ./...); done
for d in ai/services/*/; do (cd "$d" && pytest); done

echo "== 阶段4 契约校验 =="
python tools/validate_docs.py --suites yaml,json,schema,openapi

echo "== 阶段6 构建 =="
# 产物写入临时目录，不在服务目录留下二进制（与 ci.yml 一致）
tmpbin="$(mktemp -d)"
for m in services/*/; do
  # 仅含库包（无 main）的模块（如 services/region）不能配合 -o 目录构建，做编译检查即可
  if (cd "$m" && go list -f '{{.Name}}' ./... 2>/dev/null | grep -q '^main$'); then
    (cd "$m" && go build -o "$tmpbin/" ./...)
  else
    (cd "$m" && go build ./...)
  fi
done
rm -rf "$tmpbin"

echo "== 阶段5 本地等价：仓内密钥模式扫描 =="
python tools/validate_docs.py --suites secrets

echo "全部本地检查通过"
