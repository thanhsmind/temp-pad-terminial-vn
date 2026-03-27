@echo off
echo Building VN Input Helper...
go build -ldflags="-H windowsgui -s -w" -o vn-input-helper.exe main.go
if %ERRORLEVEL% EQU 0 (
    echo Build thanh cong! File: vn-input-helper.exe
    echo.
    echo App chay ngam, khong co console.
    echo Log ghi vao file: vn-input-helper.log
    echo Khi khoi dong se hien MessageBox xac nhan.
) else (
    echo Build that bai!
)
pause