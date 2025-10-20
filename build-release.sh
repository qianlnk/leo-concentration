#!/bin/bash
# Leo专注力训练系统 - 跨平台构建脚本
# 此脚本用于在macOS/Linux上构建Windows、macOS、Linux版本

set -e

echo "========================================"
echo "Leo专注力训练系统 - 发布构建工具"
echo "========================================"
echo ""

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "[错误] 未检测到Go编译器，请先安装Go语言环境"
    echo "下载地址: https://golang.org/dl/"
    exit 1
fi

echo "[1/6] 检查Go版本..."
go version
echo ""

echo "[2/6] 下载依赖..."
go mod download
echo ""

# 创建发布目录
RELEASE_DIR="release"
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

echo "[3/6] 构建Windows版本..."
GOOS=windows GOARCH=amd64 go build -o "$RELEASE_DIR/leo-concentration-windows.exe" -ldflags="-s -w" main.go
echo "✓ Windows版本构建完成 ($(ls -lh "$RELEASE_DIR/leo-concentration-windows.exe" | awk '{print $5}'))"

echo "[4/6] 构建macOS版本..."
GOOS=darwin GOARCH=amd64 go build -o "$RELEASE_DIR/leo-concentration-macos-amd64" -ldflags="-s -w" main.go
echo "✓ macOS (Intel) 版本构建完成 ($(ls -lh "$RELEASE_DIR/leo-concentration-macos-amd64" | awk '{print $5}'))"

GOOS=darwin GOARCH=arm64 go build -o "$RELEASE_DIR/leo-concentration-macos-arm64" -ldflags="-s -w" main.go
echo "✓ macOS (Apple Silicon) 版本构建完成 ($(ls -lh "$RELEASE_DIR/leo-concentration-macos-arm64" | awk '{print $5}'))"

echo "[5/6] 构建Linux版本..."
GOOS=linux GOARCH=amd64 go build -o "$RELEASE_DIR/leo-concentration-linux" -ldflags="-s -w" main.go
echo "✓ Linux版本构建完成 ($(ls -lh "$RELEASE_DIR/leo-concentration-linux" | awk '{print $5}'))"

echo "[6/6] 创建发布包..."

# Windows发布包
WIN_DIR="$RELEASE_DIR/leo-concentration-windows"
mkdir -p "$WIN_DIR/data"
cp "$RELEASE_DIR/leo-concentration-windows.exe" "$WIN_DIR/"

# 创建Windows启动脚本
cat > "$WIN_DIR/start.bat" << 'EOF'
@echo off
echo ========================================
echo Leo专注力训练系统
echo ========================================
echo.
echo 正在启动服务器...
echo 服务器地址: http://localhost:8080
echo.
echo 请在浏览器中打开上述地址使用系统
echo 按 Ctrl+C 可以停止服务器
echo.
start http://localhost:8080
leo-concentration-windows.exe
pause
EOF

cp README.md "$WIN_DIR/" 2>/dev/null || true
cd "$RELEASE_DIR" && zip -r leo-concentration-windows.zip leo-concentration-windows
cd ..
echo "✓ Windows发布包: $RELEASE_DIR/leo-concentration-windows.zip"

# macOS发布包 (Intel)
MACOS_INTEL_DIR="$RELEASE_DIR/leo-concentration-macos-intel"
mkdir -p "$MACOS_INTEL_DIR/data"
cp "$RELEASE_DIR/leo-concentration-macos-amd64" "$MACOS_INTEL_DIR/leo-concentration"
chmod +x "$MACOS_INTEL_DIR/leo-concentration"

cat > "$MACOS_INTEL_DIR/start.sh" << 'EOF'
#!/bin/bash
echo "========================================"
echo "Leo专注力训练系统"
echo "========================================"
echo ""
echo "正在启动服务器..."
echo "服务器地址: http://localhost:8080"
echo ""
echo "请在浏览器中打开上述地址使用系统"
echo "按 Ctrl+C 可以停止服务器"
echo ""
open http://localhost:8080
./leo-concentration
EOF
chmod +x "$MACOS_INTEL_DIR/start.sh"

cp README.md "$MACOS_INTEL_DIR/" 2>/dev/null || true
cd "$RELEASE_DIR" && tar -czf leo-concentration-macos-intel.tar.gz leo-concentration-macos-intel
cd ..
echo "✓ macOS Intel发布包: $RELEASE_DIR/leo-concentration-macos-intel.tar.gz"

# macOS发布包 (Apple Silicon)
MACOS_ARM_DIR="$RELEASE_DIR/leo-concentration-macos-arm"
mkdir -p "$MACOS_ARM_DIR/data"
cp "$RELEASE_DIR/leo-concentration-macos-arm64" "$MACOS_ARM_DIR/leo-concentration"
chmod +x "$MACOS_ARM_DIR/leo-concentration"

cat > "$MACOS_ARM_DIR/start.sh" << 'EOF'
#!/bin/bash
echo "========================================"
echo "Leo专注力训练系统"
echo "========================================"
echo ""
echo "正在启动服务器..."
echo "服务器地址: http://localhost:8080"
echo ""
echo "请在浏览器中打开上述地址使用系统"
echo "按 Ctrl+C 可以停止服务器"
echo ""
open http://localhost:8080
./leo-concentration
EOF
chmod +x "$MACOS_ARM_DIR/start.sh"

cp README.md "$MACOS_ARM_DIR/" 2>/dev/null || true
cd "$RELEASE_DIR" && tar -czf leo-concentration-macos-arm.tar.gz leo-concentration-macos-arm
cd ..
echo "✓ macOS ARM发布包: $RELEASE_DIR/leo-concentration-macos-arm.tar.gz"

# Linux发布包
LINUX_DIR="$RELEASE_DIR/leo-concentration-linux"
mkdir -p "$LINUX_DIR/data"
cp "$RELEASE_DIR/leo-concentration-linux" "$LINUX_DIR/leo-concentration"
chmod +x "$LINUX_DIR/leo-concentration"

cat > "$LINUX_DIR/start.sh" << 'EOF'
#!/bin/bash
echo "========================================"
echo "Leo专注力训练系统"
echo "========================================"
echo ""
echo "正在启动服务器..."
echo "服务器地址: http://localhost:8080"
echo ""
echo "请在浏览器中打开上述地址使用系统"
echo "按 Ctrl+C 可以停止服务器"
echo ""
xdg-open http://localhost:8080 2>/dev/null || echo "请手动在浏览器中打开 http://localhost:8080"
./leo-concentration
EOF
chmod +x "$LINUX_DIR/start.sh"

cp README.md "$LINUX_DIR/" 2>/dev/null || true
cd "$RELEASE_DIR" && tar -czf leo-concentration-linux.tar.gz leo-concentration-linux
cd ..
echo "✓ Linux发布包: $RELEASE_DIR/leo-concentration-linux.tar.gz"

echo ""
echo "========================================"
echo "构建完成！"
echo "========================================"
echo ""
echo "发布文件:"
ls -lh "$RELEASE_DIR"/*.zip "$RELEASE_DIR"/*.tar.gz 2>/dev/null || true
echo ""
echo "使用方法:"
echo "  Windows: 解压 leo-concentration-windows.zip，双击 start.bat"
echo "  macOS:   解压对应的 tar.gz，运行 ./start.sh"
echo "  Linux:   解压 leo-concentration-linux.tar.gz，运行 ./start.sh"
echo ""
