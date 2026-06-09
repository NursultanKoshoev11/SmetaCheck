package api

import(
 "errors"
 "mime/multipart"
 "path/filepath"
)

func ValidateUpload(h *multipart.FileHeader)error{
 if h.Size<=0{return errors.New("empty file")}
 if h.Size>25*1024*1024{return errors.New("file too large")}
 ext:=filepath.Ext(h.Filename)
 if ext!=".xlsx"&&ext!=".xls"&&ext!=".pdf"&&ext!=".docx"{return errors.New("unsupported file")}
 return nil
}
