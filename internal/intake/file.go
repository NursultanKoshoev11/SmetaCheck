package intake

type FileInfo struct{
 Name string
 Path string
 SizeMB int
 Mime string
}

func AllowedExt(ext string)bool{
 return ext==".xlsx"||ext==".xls"||ext==".pdf"||ext==".docx"
}
