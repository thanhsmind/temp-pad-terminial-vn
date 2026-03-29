# VN Input Helper

App nhỏ giúp nhập tiếng Việt vào terminal/app bất kỳ trên Windows mà không bị lỗi mất chữ.

## Cách hoạt động

1. Chạy app → app nằm im trong background
2. Nhấn phím tắt (mặc định `Ctrl+Shift+Space`) → popup hiện ra
3. Gõ tiếng Việt thoải mái trong popup (hỗ trợ IME native của Windows)
4. `Ctrl+Enter` hoặc nhấn OK → nội dung được paste vào đúng chỗ con trỏ trước đó
5. `Esc` → hủy

## Build

Yêu cầu: Go 1.21+   

```bash
go build -ldflags="-H windowsgui -s -w" -o vn-input-helper.exe main.go
```

Hoặc chạy `build.bat`.

Flag `-H windowsgui` để không hiện console window khi chạy.
Nếu muốn debug, build không có flag này:

```bash
go build -o vn-input-helper-debug.exe main.go
```

## Cấu hình

Chỉnh file `config.json` cùng thư mục với exe:

```json
{
  "hotkey": {
    "modifiers": ["Ctrl", "Shift"],
    "key": "Space"
  },
  "window": {
    "width": 500,
    "height": 200,
    "title": "Nhập nội dung"
  }
}
```

### Phím tắt hỗ trợ

**Modifiers:** `Ctrl`, `Shift`, `Alt`, `Win`

**Key:**

- Chữ cái: `A`-`Z`
- Số: `0`-`9`
- Đặc biệt: `Space`, `Tab`, `Enter`
- F-keys: `F1`-`F24`

### Ví dụ cấu hình

```json
// Alt+Space
{"modifiers": ["Alt"], "key": "Space"}

// Ctrl+Alt+V
{"modifiers": ["Ctrl", "Alt"], "key": "V"}

// Win+`  (backtick không hỗ trợ, dùng key khác)
{"modifiers": ["Win"], "key": "Q"}
```

## Lưu ý

- App tự backup clipboard trước khi paste và restore lại sau 500ms
- Popup luôn hiện trên cùng (topmost) để không bị che
- Dùng Win32 EDIT control native → IME tiếng Việt hoạt động bình thường
- Nếu phím tắt bị conflict với app khác, đổi trong config.json rồi restart

