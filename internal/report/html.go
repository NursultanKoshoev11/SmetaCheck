package report

import "strings"

func HTML(title,summary string,items []string)string{
 var b strings.Builder
 b.WriteString("<html><body><h1>"); b.WriteString(title); b.WriteString("</h1>")
 b.WriteString("<p>"); b.WriteString(summary); b.WriteString("</p><ul>")
 for _,v:=range items{b.WriteString("<li>"); b.WriteString(v); b.WriteString("</li>")}
 b.WriteString("</ul></body></html>")
 return b.String()
}
