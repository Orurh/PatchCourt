package check

func renderCheckHTMLTemplate(jsonPayload string) string {
	return checkHTMLDocumentStart +
		checkHTMLCSS +
		checkHTMLAfterCSSBeforeData +
		jsonPayload +
		checkHTMLAfterDataBeforeJS +
		checkHTMLJS +
		checkHTMLDocumentEnd
}
