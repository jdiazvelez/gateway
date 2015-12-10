package mail

import (
	"fmt"
)

const mailTemplate = `From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";

<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
 <head>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
	<title>{{.Subject}}</title>
	<style type="text/css">
	 body{width:100% !important; -webkit-text-size-adjust:100%; -ms-text-size-adjust:100%; margin:0; padding:0;}
	</style>
 </head>
 <body>{{template "body" .}}</body>
</html>
`

type EmailTemplate struct {
	From    string
	To      string
	Subject string
	Scheme  string
	Host    string
	Port    int64
	Prefix  string
	Token   string
}

func (e *EmailTemplate) UrlPrefix() string {
	if (e.Scheme == "http" && e.Port == 80) || (e.Scheme == "https" && e.Port == 443) {
		return fmt.Sprintf("%v://%v%v", e.Scheme, e.Host, e.Prefix)
	} else {
		return fmt.Sprintf("%v://%v:%v%v", e.Scheme, e.Host, e.Port, e.Prefix)
	}
}