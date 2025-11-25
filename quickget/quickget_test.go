package quickget

import (
	"log"
	"strings"
	"testing"
)

func TestQuickGetPath(t *testing.T) {
	// Test that the path to the quickget script is correct
	path, err := CreateQuickGet()
	if err != nil {
		t.Fatalf("Error creating quickget script: %v", err)
	}
	log.Println("path=", path)
	if path == "" {
		t.Errorf("Path to quickget script is empty")
	}
}

func TestParseLastURL(t *testing.T) {
	input := `
Downloading Windows 11 (English International)   
 - Parsing download page: https://www.microsoft.com/en-us/software-download/windows11
 - Getting Product edition ID: 3113             
 - Permit Session ID: 6e64fd31-3ed0-4450-a33d-90c04a4d2378                                                             
 - Getting language SKU ID: 18481                                                                                           
 - Getting ISO download link...              
 - URL: https://software.download.prss.microsoft.com/dbazure/Win11_24H2_EnglishInternational_x64.iso
windows-11:                          https://software.download.prss.microsoft.com/dbazure/Win11_24H2_EnglishInternational_x64.is
o?t=accb893f-365a-4492-af0d-c6e5cc12e150&P1=1748068024&P2=601&P3=2&P4=M9iVhLiSQLdZBN8xE%2fLVpuMMeWN7nFs9dYdxHzqSGrAD0dFJlBn5BmfH
Rax3J1SGZ1aXo%2f7y6UeqyiIbRrn8e5yly%2b%2bi1ml9%2fdEQBbwJfuMSeZ%2bys824Mm5O4ugyJvdhu5%2b5%2f1PXE3C7Tj0I4IAoUjD09XkjI8BLmRfTKu5REq
V0xEJvQFablM%2bByEQ%2bAWidLloEveLSg7vIZ8xaY8HLLd8mXWnworHlrP3jw0s%2bEltSZHh8gpBFwGz4pBx1iI3i%2bDiO67cJAJL5gCGkqLt5WfEeGMHjSJVWwU
CZE9CIpzi5npeJ1d6uDDZnj9LEc1HPeWfbUToKSqI0GRThSJDAiQ%3d%3d
	`
	input = strings.Replace(input, "\n", "", -1)
	log.Println("input=", input)
	urlStr, err := ParseLastURL(input)
	if err != nil {
		t.Fatal(err)
	}
	log.Println("urlStr=", urlStr)
}
