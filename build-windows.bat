@echo off
REM Leo专注力训练系统 - Windows构建脚本
REM 此脚本用于在Windows系统上构建可执行文件

set VERSION=v0.0.3

echo ========================================
echo Leo专注力训练系统 %VERSION% - Windows构建工具
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

echo [3/4] 构建Windows可执行文件 (%VERSION%)...
set GOOS=windows
set GOARCH=amd64
go build -o leo-concentration.exe -ldflags="-s -w -X main.Version=%VERSION%" main.go
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 构建失败
    pause
    exit /b 1
)
echo.

echo [4/4] 创建发布包...
set RELEASE_DIR=leo-concentration-%VERSION%-windows-amd64
if exist "%RELEASE_DIR%" rmdir /s /q "%RELEASE_DIR%"
mkdir "%RELEASE_DIR%"
mkdir "%RELEASE_DIR%\data"

REM 复制可执行文件
copy leo-concentration.exe "%RELEASE_DIR%\"
echo 已复制: leo-concentration.exe

REM 创建启动脚本
echo @echo off > "%RELEASE_DIR%\start.bat"
echo echo ======================================== >> "%RELEASE_DIR%\start.bat"
echo echo Leo专注力训练系统 %VERSION% >> "%RELEASE_DIR%\start.bat"
echo echo ======================================== >> "%RELEASE_DIR%\start.bat"
echo echo. >> "%RELEASE_DIR%\start.bat"
echo echo 正在启动服务器... >> "%RELEASE_DIR%\start.bat"
echo echo 服务器地址: http://localhost:8080 >> "%RELEASE_DIR%\start.bat"
echo echo. >> "%RELEASE_DIR%\start.bat"
echo echo 请在浏览器中打开上述地址使用系统 >> "%RELEASE_DIR%\start.bat"
echo echo 按 Ctrl+C 可以停止服务器 >> "%RELEASE_DIR%\start.bat"
echo echo. >> "%RELEASE_DIR%\start.bat"
echo start http://localhost:8080 >> "%RELEASE_DIR%\start.bat"
echo leo-concentration.exe >> "%RELEASE_DIR%\start.bat"
echo pause >> "%RELEASE_DIR%\start.bat"

echo 已创建: start.bat

REM 创建使用说明
echo Leo专注力训练系统 %VERSION% > "%RELEASE_DIR%\使用说明.txt"
echo. >> "%RELEASE_DIR%\使用说明.txt"
echo 使用方法: >> "%RELEASE_DIR%\使用说明.txt"
echo 1. 双击 start.bat 启动系统 >> "%RELEASE_DIR%\使用说明.txt"
echo 2. 浏览器会自动打开 http://localhost:8080 >> "%RELEASE_DIR%\使用说明.txt"
echo 3. 如果浏览器未自动打开，请手动访问 http://localhost:8080 >> "%RELEASE_DIR%\使用说明.txt"
echo 4. 按 Ctrl+C 可以停止服务器 >> "%RELEASE_DIR%\使用说明.txt"
echo. >> "%RELEASE_DIR%\使用说明.txt"
echo 更新日志 (v0.0.3): >> "%RELEASE_DIR%\使用说明.txt"
echo - 平衡超人改为正向计时，移除倒计时和欢迎页面 >> "%RELEASE_DIR%\使用说明.txt"
echo - 平衡超人支持左右脚独立训练和统计 >> "%RELEASE_DIR%\使用说明.txt"
echo - Chart.js 图表库改为本地引用，离线可用 >> "%RELEASE_DIR%\使用说明.txt"
echo. >> "%RELEASE_DIR%\使用说明.txt"
echo 更新日志 (v0.0.2): >> "%RELEASE_DIR%\使用说明.txt"
echo - 新增小鱼历险记游戏，包含8个难度级别 >> "%RELEASE_DIR%\使用说明.txt"
echo - 添加音效系统（吃鱼音效、规则变化提示音、背景音乐） >> "%RELEASE_DIR%\使用说明.txt"
echo - 优化游戏控制（点击响应、鱼头方向、碰撞检测） >> "%RELEASE_DIR%\使用说明.txt"
echo - 错误的鱼会追逐玩家，增加游戏难度 >> "%RELEASE_DIR%\使用说明.txt"
echo - 智能生成符合规则的鱼，提高可玩性 >> "%RELEASE_DIR%\使用说明.txt"
echo - 全屏游戏界面，规则用红色大字展示 >> "%RELEASE_DIR%\使用说明.txt"
echo. >> "%RELEASE_DIR%\使用说明.txt"
echo 数据存储: >> "%RELEASE_DIR%\使用说明.txt"
echo - 所有训练数据保存在 data\leo_concentration.db >> "%RELEASE_DIR%\使用说明.txt"
echo. >> "%RELEASE_DIR%\使用说明.txt"

echo 已创建: 使用说明.txt

REM 打包成zip（使用PowerShell）
echo.
echo [5/5] 打包压缩文件...
powershell -Command "Compress-Archive -Path '%RELEASE_DIR%' -DestinationPath '%RELEASE_DIR%.zip' -Force"
if %ERRORLEVEL% EQU 0 (
    echo 已创建: %RELEASE_DIR%.zip
) else (
    echo [警告] 压缩失败，但文件夹已创建
)

echo.
echo ========================================
echo 构建完成！
echo ========================================
echo.
echo 发布文件:
echo - 文件夹: %RELEASE_DIR%\
echo - 压缩包: %RELEASE_DIR%.zip
echo.
echo Windows用户使用方法:
echo 1. 解压 %RELEASE_DIR%.zip
echo 2. 进入文件夹，双击 start.bat 启动系统
echo 3. 浏览器会自动打开 http://localhost:8080
echo.
pause
