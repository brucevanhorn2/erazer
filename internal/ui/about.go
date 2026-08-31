package ui

import (
	"fmt"
	"strings"
)

// logo is a "bigmono12"-style ASCII rendering of "erazer" (via
// `toilet -f bigmono12 erazer`), colored at render time with a gradient
// instead of baking in ANSI codes here — same approach as exfil's About
// screen.
const logo = `  ░████▒    ██░████   ▒████▓   ████████   ░████▒    ██░████ 
 ░██████▒   ███████   ██████▓  ████████  ░██████▒   ███████ 
 ██▒  ▒██   ███░      █▒  ▒██      ▒██▒  ██▒  ▒██   ███░    
 ████████   ██         ▒█████     ▒██▒   ████████   ██      
 ████████   ██       ░███████    ▒██▒    ████████   ██      
 ██         ██       ██▓░  ██   ▒██▒     ██         ██      
 ███░  ▒█   ██       ██▒  ███  ▒██▒      ███░  ▒█   ██      
 ░███████   ██       ████████  ████████  ░███████   ██      
  ░█████▒   ██        ▓███░██  ████████   ░█████▒   ██      `

// logoFrom and logoTo are the gradient endpoints for the logo: cyan fading
// to purple, matching exfil's About screen exactly.
const (
	logoFrom = "#00E5FF"
	logoTo   = "#B341F5"
	tagline  = "secure delete for people who've had a bad week"
	version  = "dev"
)

// AboutPane renders erazer's About screen: logo, tagline, version, license.
type AboutPane struct {
	Width  int
	Height int
}

func NewAboutPane() *AboutPane {
	return &AboutPane{}
}

func (a *AboutPane) View(theme Theme) string {
	lines := []string{
		gradientLogo(logo, logoFrom, logoTo),
		"",
		theme.BrowserFile.Render(tagline),
		"",
		theme.BrowserDir.Render(fmt.Sprintf("%-10s", "Version:")) + version,
		theme.BrowserDir.Render(fmt.Sprintf("%-10s", "License:")) + "MIT",
		theme.BrowserDir.Render(fmt.Sprintf("%-10s", "Source:")) + "github.com/brucevanhorn2/erazer",
		"",
		theme.StatusKey.Render("press any key to close"),
	}
	content := strings.Join(lines, "\n")
	return gradientBox(content, a.Width, a.Height-2, theme.PrimaryColor, theme.SecondaryColor)
}
