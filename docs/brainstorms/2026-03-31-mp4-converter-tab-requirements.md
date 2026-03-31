---
date: 2026-03-31
topic: mp4-converter-tab
---

# MP4 Converter Tab

## Problem Frame
Users cần convert file MP4 sang WebM (best quality) hoặc MP3 ngay trên máy local, không muốn dùng tool online. App VN Input Helper đã chạy thường trực nên thêm tính năng này vào cùng popup là tiện lợi.

## Requirements

**UI & Navigation**
- R1. Thêm Win32 Tab Control (SysTabControl32) vào popup window hiện tại: Tab 1 = VN Input (giữ nguyên), Tab 2 = Video Convert
- R2. Tab Video Convert có: nút "Chọn file MP4" mở Open File dialog (filter *.mp4), hiển thị tên file đã chọn, các nút convert

**Convert Options**
- R3. Option 1: Convert MP4 sang WebM VP9 với chất lượng tốt nhất (VP9 video codec, CRF tối ưu - giá trị cụ thể xác định khi planning)
- R4. Option 2: Convert MP4 sang MP4 H.265/HEVC (re-encode video với H.265 codec, chất lượng tốt nhất)
- R5. Option 3: Convert MP4 sang MP3 (extract audio, fixed preset 320kbps)
- R6. Mỗi option là một button riêng. User chọn file trước, rồi bấm button convert mong muốn

**FFmpeg**
- R7. Đặt ffmpeg.exe cạnh file exe chính (không embed vào binary). App tìm ffmpeg.exe trong cùng thư mục và gọi trực tiếp
- R8. Nếu không tìm thấy ffmpeg.exe, hiện thông báo lỗi: "Không tìm thấy ffmpeg.exe. Vui lòng đặt ffmpeg.exe cạnh vn-input-helper.exe"

**Progress & Output**
- R9. Hiển thị progress bar + phần trăm hoàn thành trong quá trình convert (parse từ FFmpeg stderr output). Nếu không xác định được tổng duration, hiển thị indeterminate progress bar (marquee mode)
- R10. Mở dialog Save As **trước khi** bắt đầu convert để user chọn nơi lưu (tên mặc định = basename file gốc + extension mới). FFmpeg ghi trực tiếp vào đường dẫn đã chọn
- R11. Hiển thị trạng thái "Hoàn thành!" khi xong. Nếu thất bại: hiện thông báo lỗi và xóa file output không hoàn chỉnh (nếu có)

## Success Criteria
- User có thể chọn file MP4 và convert sang WebM hoặc MP3 thành công
- Progress bar cập nhật realtime trong quá trình convert
- File output được lưu đúng nơi user chọn
- Không ảnh hưởng đến tính năng VN Input hiện tại ở Tab 1

## Scope Boundaries
- Chỉ hỗ trợ input MP4, không hỗ trợ các format khác
- Chỉ 3 output format: WebM VP9, MP4 H.265, và MP3
- Không hỗ trợ batch convert (chỉ 1 file mỗi lần)
- Không có tùy chọn nâng cao cho user - dùng fixed preset (VP9 best quality cho WebM, 320kbps cho MP3)
- FFmpeg đặt cạnh exe, không embed và không tự download

## Key Decisions
- **Bundle FFmpeg cạnh exe**: Chấp nhận distribution nặng hơn (~80-100MB) để user không cần cài gì thêm. ffmpeg.exe đặt cùng thư mục, không embed vào binary
- **Tab trong popup hiện tại**: Tận dụng window đã có thay vì tạo window/process riêng
- **Save As trước khi convert**: Cho user chọn nơi lưu trước, FFmpeg ghi trực tiếp. Tránh tình huống convert xong nhưng không lưu được
- **Progress bar với marquee fallback**: Parse FFmpeg output để hiển thị %, fallback sang marquee mode khi không biết duration

## Outstanding Questions

### Deferred to Planning
- [Affects R1][Technical] Cần init `comctl32.dll` (`InitCommonControlsEx` với `ICC_TAB_CLASSES`) trước khi tạo SysTabControl32. Cần thêm `comdlg32.dll` cho file dialogs (`GetOpenFileNameW`, `GetSaveFileNameW`)
- [Affects R1][Technical] Quản lý visibility child controls giữa 2 tab: cần xử lý `WM_NOTIFY`/`TCN_SELCHANGE` (chưa có trong codebase hiện tại) để show/hide nhóm controls tương ứng
- [Affects R9][Technical] FFmpeg phải chạy trong goroutine riêng (không block UI thread). Cần `PostMessageW` với custom message ID (WM_APP+N) để gửi progress update từ goroutine về UI thread
- [Affects R9][Needs research] Parse FFmpeg progress output format (`duration` + `time=`) để tính phần trăm chính xác
- [Affects R3][Needs research] FFmpeg command tối ưu cho VP9 WebM best quality (CRF value, speed preset, audio codec)
- [Affects R4][Needs research] FFmpeg command cho H.265/HEVC encoding best quality (CRF value, preset, có cần libx265 hay dùng built-in encoder)

## Next Steps
-> `/ce:plan` for structured implementation planning
