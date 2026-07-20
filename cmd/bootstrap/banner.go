package bootstrap

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
)

// GenerateBannerWithDoomFont generates ASCII art banner using Doom font and saves to file
func GenerateBannerWithDoomFont(text, filename string) error {
	// Try to generate using online figlet API first
	banner, err := generateBannerFromAPI(text)
	if err != nil {
		// Fallback to local Doom font implementation
		fmt.Printf("API call failed, using local Doom font implementation: %v\n", err)
		banner, err = generateBannerWithLocalDoom(text)
		if err != nil {
			return fmt.Errorf("failed to generate banner: %w", err)
		}
	}
	// Save to file
	return os.WriteFile(filename, []byte(banner), pkgconst.BannerFilePerm)
}

// generateBannerFromAPI tries to generate banner using online figlet API
func generateBannerFromAPI(text string) (string, error) {
	// Use patorjk.com API - convert text to URL encoded format
	encodedText := url.QueryEscape(text)
	apiURL := fmt.Sprintf(pkgconst.BannerAPIURLTemplate, encodedText)

	client := &http.Client{
		Timeout: pkgconst.BannerAPITimeout,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	// Set headers to mimic browser request
	req.Header.Set("User-Agent", pkgconst.BannerUserAgent)
	req.Header.Set("Accept", pkgconst.BannerAcceptHeader)
	req.Header.Set("Referer", pkgconst.BannerRefererURL)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Check if response is HTML (API error)
	bodyStr := string(body)
	if strings.Contains(bodyStr, pkgconst.HTMLDoctypePrefix) || strings.Contains(bodyStr, pkgconst.HTMLTagPrefix) || strings.Contains(bodyStr, pkgconst.HTML404Error) {
		return "", fmt.Errorf("API returned HTML error page instead of ASCII art")
	}
	// The API returns plain text ASCII art
	banner := strings.TrimSpace(bodyStr)
	if banner == "" {
		return "", fmt.Errorf("empty response from API")
	}
	// Clean up the response - remove any HTML tags if present
	banner = strings.ReplaceAll(banner, pkgconst.HTMLBrTag, "\n")
	banner = strings.ReplaceAll(banner, pkgconst.HTMLBrSelfClose, "\n")
	banner = strings.ReplaceAll(banner, pkgconst.HTMLBrCloseSpace, "\n")
	// Verify it's actually ASCII art (should contain common ASCII art characters)
	if !strings.ContainsAny(banner, pkgconst.ASCIIArtChars) {
		return "", fmt.Errorf("API response doesn't appear to be ASCII art")
	}
	return banner, nil
}

// generateBannerWithLocalDoom generates banner using local Doom font implementation
func generateBannerWithLocalDoom(text string) (string, error) {
	// Doom font character mappings
	doomChars := getDoomFontChars()
	text = strings.ToUpper(text)
	lines := make([]string, pkgconst.DoomFontHeight)
	for _, char := range text {
		if charArt, ok := doomChars[char]; ok {
			charLines := strings.Split(charArt, "\n")
			// Remove empty lines at the end
			for len(charLines) > 0 && strings.TrimSpace(charLines[len(charLines)-1]) == "" {
				charLines = charLines[:len(charLines)-1]
			}
			// Find the maximum width of this character
			maxWidth := 0
			for _, line := range charLines {
				if len(line) > maxWidth {
					maxWidth = len(line)
				}
			}
			// Pad lines to ensure consistent width
			for i := 0; i < pkgconst.DoomFontHeight; i++ {
				if i < len(charLines) {
					// Pad the line to maxWidth
					paddedLine := charLines[i]
					for len(paddedLine) < maxWidth {
						paddedLine += " "
					}
					lines[i] += paddedLine
				} else {
					// Add empty line with same width
					lines[i] += strings.Repeat(" ", maxWidth)
				}
			}
		} else if char == ' ' {
			// Add space (DoomFontSpaceWidth characters wide for Doom font)
			for i := 0; i < pkgconst.DoomFontHeight; i++ {
				lines[i] += strings.Repeat(" ", pkgconst.DoomFontSpaceWidth)
			}
		} else {
			// Unknown character, use placeholder
			for i := 0; i < pkgconst.DoomFontHeight; i++ {
				lines[i] += pkgconst.DoomFontUnknownChar
			}
		}
	}
	// Remove trailing empty lines and trim each line
	result := strings.Join(lines, "\n")
	return strings.TrimRight(result, "\n"), nil
}

// getDoomFontChars returns a map of characters to their ASCII art representation
// This uses the Doom font style from patorjk.com
func getDoomFontChars() map[rune]string {
	return map[rune]string{
		'L': ` _    
| |   
| |   
| |
| |___ 
|_____|`,
		'I': ` _ 
| |
| |
| |
| |
|_|`,
		'N': ` _   _ 
| \ | |
|  \| |
| . \ |
| |\  |
| | \ |`,
		'G': ` _____ 
|  __ \
| |  \/
| | __ 
| |_\ \
 \____/
       
       `,
		'S': ` _____ 
/  ___|
\ --. 
  --. \
/\__/ /
\____/ 
       
       `,
		'T': ` ______
|_   _|
  | |  
  | |
  | |  
  \_/  `,
		'O': ` _____ 
|  _  |
| | | |
| | | |
| |_| |
\_____/`,
		'R': ` ______ 
| ___ \
| |_/ /
|    / 
| |\ \ 
\_| \_|`,
		'A': `  ___  
 / _ \ 
/ /_\ \
|  _  |
| | | |
\_| |_/
       
       `,
		'E': ` ______
|  ____|
| |__   
|  __|  
| |____ 
|______|`,
		'Y': ` __   __
\ \ / /
 \ V / 
  \ /  
  | |  
  \_/  `,
		'H': ` _   _ 
| | | |
| |_| |
|  _  |
| | | |
|_| |_|`,
		'P': ` ______ 
| ___ \
| |_/ /
|  __/ 
| |    
|_|    `,
		'C': ` _____ 
/  __ \
| /  \/
| |    
| \__/\
 \____/`,
		'D': ` ______ 
| ___ \
| | | |
| | | |
| |_| |
\____/ `,
		'F': ` ______
|  ____|
| |__   
|  __|  
| |     
|_|     `,
		'U': ` _   _ 
| | | |
| | | |
| | | |
| |_| |
 \___/ `,
		'V': `__      __
\ \    / /
 \ \  / / 
  \ \/ /  
   \  /   
    \/    `,
		'W': `__      __
\ \ /\ / /
 \ V  V / 
  \_/\_/  `,
		'X': `__   __
\ \ / /
 \ V / 
 /   \ 
/ /^\ \
\/   \/`,
		'Z': ` ______
|___  /
   / / 
  / /  
./ /___
\_____/`,
		'B': ` ______ 
| ___ \
| |_/ /
| ___ \ 
| |_/ /
\____/ `,
		'J': `     __
    / /
    | |
    | |
/\__/ /
\____/ `,
		'K': ` _   __
| | / /
| |/ / 
|    \ 
| |\  \
|_| \_|`,
		'M': ` _   _
|  \| |
| . \ |
| |\  |
|_| \_|`,
		'Q': ` _____ 
|  _  |
| | | |
| | | |
| |_\ |
 \___\`,
		' ': `   
   
   
   
   
   
   
   `,
		'0': ` _____ 
|  _  |
| | | |
| | | |
| |_| |
\_____/`,
		'1': `  __
 /  |
| | 
| | 
| | 
| | 
|_| `,
		'2': ` _____ 
/ __  \
| |  \/
| | __ 
| |_\ \
 \____/`,
		'3': ` _____ 
|____ |
    / /
    \ \
.___/ /
\____/ `,
		'4': `   ___ 
  /   |
 / /| |
/ /_| |
\___  |
    |_|`,
		'5': ` ______
|____ |
    / /
    \ \
.___/ /
\____/ `,
		'6': `  ____ 
 / ___|
/ /___ 
| ___ \
| \_/ |
\_____/`,
		'7': ` ______
|___  /
   / / 
  / /  
./ /   
\_/    `,
		'8': ` _____ 
|  _  |
 \ V / 
 / _ \ 
| |_| |
\_____/`,
		'9': ` _____ 
|  _  |
| |_| |
\____ |
.___/ /
\____/ `,
	}
}

// EnsureBannerFile ensures banner.txt exists, generates it if it doesn't
func EnsureBannerFile(filename, defaultText string) error {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// File doesn't exist, generate it
		if defaultText == "" {
			defaultText = pkgconst.DefaultBannerText
		}
		fmt.Printf("Banner file not found, generating %s with Doom font...\n", filename)
		err := GenerateBannerWithDoomFont(defaultText, filename)
		if err != nil {
			return fmt.Errorf("failed to generate banner file: %w", err)
		}
		fmt.Printf("Banner file generated successfully: %s\n", filename)
	}
	return nil
}
