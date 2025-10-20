@echo off
REM Leo专注力训练系统 - Windows构建脚本
REM 此脚本用于在Windows系统上构建可执行文件

echo ========================================
echo Leo专注力训练系统 - Windows构建工具
echo ========================================
echo.

REM 检查Go是否安装
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 未检测到Go编译器，请先安装Go语言环境
    echo 下载地址: https://golang.org/dl/
    pause
    exit /b 1
)

echo [1/4] 检查Go版本...
go version
echo.

echo [2/4] 下载依赖...
go mod download
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 依赖下载失败
    pause
    exit /b 1
)
echo.

echo [3/4] 构建Windows可执行文件...
set GOOS=windows
set GOARCH=amd64
go build -o leo-concentration.exe -ldflags="-s -w" main.go
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 构建失败
    pause
    exit /b 1
)
echo.

echo [4/4] 创建发布包...
if not exist "release" mkdir release
if not exist "release\data" mkdir release\data

REM 复制可执行文件
copy leo-concentration.exe release\
echo 已复制: leo-concentration.exe

REM 创建启动脚本
echo @echo off > release\start.bat
echo echo ======================================== >> release\start.bat
echo echo Leo专注力训练系统 >> release\start.bat
echo echo ======================================== >> release\start.bat
echo echo. >> release\start.bat
echo echo 正在启动服务器... >> release\start.bat
echo echo 服务器地址: http://localhost:8080 >> release\start.bat
echo echo. >> release\start.bat
echo echo 请在浏览器中打开上述地址使用系统 >> release\start.bat
echo echo 按 Ctrl+C 可以停止服务器 >> release\start.bat
echo echo. >> release\start.bat
echo start http://localhost:8080 >> release\start.bat
echo leo-concentration.exe >> release\start.bat
echo pause >> release\start.bat

echo 已创建: start.bat

REM 复制说明文件（如果存在）
if exist "README.md" copy README.md release\
if exist "CLAUDE.md" copy CLAUDE.md release\项目说明.md

echo.
echo ========================================
echo 构建完成！
echo ========================================
echo.
echo 发布文件位于: release\ 目录
echo.
echo 使用方法:
echo 1. 将 release 文件夹复制到目标计算机
echo 2. 双击 start.bat 启动系统
echo 3. 浏览器会自动打开 http://localhost:8080
echo.
pause
