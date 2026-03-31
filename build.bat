@echo off
echo Building VN Input Helper...
go build -ldflags="-H windowsgui -s -w" -o vn-input-helper.exe main.go
if %ERRORLEVEL% EQU 0 (
    echo Build thanh cong! File: vn-input-helper.exe
    echo.
    echo App chay ngam, khong co console.
    echo Log ghi vao file: vn-input-helper.log
    echo Khi khoi dong se hien MessageBox xac nhan.
    echo.
    echo LUU Y: De su dung tinh nang Video Convert, dat ffmpeg.exe
    echo canh file vn-input-helper.exe.
    echo Download ffmpeg tai: https://www.gyan.dev/ffmpeg/builds/
    echo ^(chon ban "full" de co day du codec^)
) else (
    echo Build that bai!
)
pause