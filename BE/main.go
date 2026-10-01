package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"github.com/jung-kurt/gofpdf/v2"
)

type InvoiceItem struct {
	SlNo        int     `json:"slNo"`
	Particulars string  `json:"particulars"`
	HSNCode     string  `json:"hsnCode"`
	QTY         float64 `json:"qty"`
	Rate        float64 `json:"rate"`
	Amount      float64 `json:"amount"`
}

type InvoiceRequest struct {
	InvoiceNo          string        `json:"invoiceNo"`
	InvoiceDate        string        `json:"invoiceDate"`
	ReceiverName       string        `json:"receiverName"`
	ReceiverAddress    string        `json:"receiverAddress"`
	ReceiverGSTIN      string        `json:"receiverGstin"`
	ReceiverState      string        `json:"receiverState"`
	StateCode          string        `json:"stateCode"`
	Items              []InvoiceItem `json:"items"`
	CGSTRate           float64       `json:"cgstRate"`
	SGSTRate           float64       `json:"sgstRate"`
	IGSTRate           float64       `json:"igstRate"`
	RoundOff           float64       `json:"roundOff"`
	TotalInvoiceValue  float64       `json:"totalInvoiceValue"`
	TotalAmountInWords string        `json:"totalAmountInWords"`
}

type InvoiceResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	FilePath string `json:"filePath"`
}

func main() {
	// Detect user Home directory and set output directory to Downloads
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to detect user home directory: %v", err)
	}
	outputDir := filepath.Join(home, "Downloads")
	
	// Ensure the directory exists (it should, but safety first)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to verify output directory: %v", err)
	}

	http.HandleFunc("/api/invoice", handleInvoice(outputDir))

	port := "8080"
	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func enableCORS(w *http.ResponseWriter, r *http.Request) bool {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	
	if r.Method == "OPTIONS" {
		(*w).WriteHeader(http.StatusOK)
		return true
	}
	return false
}

func handleInvoice(outputDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if enableCORS(&w, r) {
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req InvoiceRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Perform Server-Side calculations to verify
		var rawSubtotal float64
		for i, item := range req.Items {
			calculatedAmount := item.QTY * item.Rate
			req.Items[i].Amount = calculatedAmount
			rawSubtotal += calculatedAmount
		}

		cgstVal := math.Round((rawSubtotal*(req.CGSTRate/100.0))*100) / 100
		sgstVal := math.Round((rawSubtotal*(req.SGSTRate/100.0))*100) / 100
		igstVal := math.Round((rawSubtotal*(req.IGSTRate/100.0))*100) / 100
		
		subtotalWithTax := rawSubtotal + cgstVal + sgstVal + igstVal
		finalValue := subtotalWithTax + req.RoundOff

		// Save PDF filename
		sanitizedName := strings.ReplaceAll(req.ReceiverName, " ", "_")
		sanitizedName = strings.ReplaceAll(sanitizedName, "/", "-")
		sanitizedName = strings.ReplaceAll(sanitizedName, "\\", "-")
		
		fileName := fmt.Sprintf("INV_%s_%s.pdf", req.InvoiceNo, sanitizedName)
		filePath := filepath.Join(outputDir, fileName)

		// Generate PDF
		err := generatePDF(req, rawSubtotal, cgstVal, sgstVal, igstVal, finalValue, filePath)
		if err != nil {
			log.Printf("PDF generation failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(InvoiceResponse{
				Success: false,
				Message: "Failed to generate PDF: " + err.Error(),
			})
			return
		}

		absPath, _ := filepath.Abs(filePath)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InvoiceResponse{
			Success:  true,
			Message:  "Invoice PDF generated and saved successfully!",
			FilePath: absPath,
		})
	}
}

func generatePDF(req InvoiceRequest, subtotal, cgstVal, sgstVal, igstVal, finalValue float64, filePath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	// Outer border for the invoice
	pdf.SetLineWidth(0.4)
	pdf.Rect(10, 10, 190, 277, "D")

	// Header line
	pdf.SetFont("Arial", "U", 12)
	pdf.CellFormat(190, 6, "TAX INVOICE", "0", 1, "C", false, 0, "")

	// Cell Phone Numbers (Top Right)
	pdf.SetFont("Arial", "", 8)
	pdf.Text(155, 14, "Cell : 7386388032")
	pdf.Text(162, 18, ": 9440597629")

	// Company Title
	pdf.Ln(4)
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(190, 8, "RAVINDRA SRI VENGAMAMBA", "0", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 8, "EQUIPMENTS & RENTALS", "0", 1, "C", false, 0, "")

	// Subtext Address
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(190, 4, "Flat No. 4-193/2, Opp : SSIT College, Main Road, B. Gangaram - 507 303,", "0", 1, "C", false, 0, "")
	pdf.CellFormat(190, 4, "Khammam Dist, Telangana.", "0", 1, "C", false, 0, "")
	
	// GST Info
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 5, "GST No. 36ANBPM4958K1ZK", "0", 1, "C", false, 0, "")

	// Invoice Number & Date Grid Section
	pdf.SetLineWidth(0.2)
	pdf.Line(10, 46, 200, 46) // Horizontal line under header info

	pdf.SetFont("Arial", "B", 10)
	pdf.SetXY(12, 47)
	pdf.CellFormat(90, 6, "Invoice No. : "+req.InvoiceNo, "0", 0, "L", false, 0, "")
	pdf.CellFormat(98, 6, "Invoice Date : "+req.InvoiceDate, "0", 1, "L", false, 0, "")

	pdf.Line(10, 53, 200, 53) // Horizontal line under Inv details

	// Details of Receiver Section
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 5, "Details of Receiver", "B", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.Ln(1)
	pdf.SetX(12)
	pdf.CellFormat(186, 5, "Name : "+req.ReceiverName, "0", 1, "L", false, 0, "")
	pdf.SetX(12)
	pdf.CellFormat(186, 5, "Address : "+req.ReceiverAddress, "0", 1, "L", false, 0, "")
	
	pdf.SetX(12)
	pdf.CellFormat(80, 5, "GSTIN : "+req.ReceiverGSTIN, "0", 0, "L", false, 0, "")
	pdf.CellFormat(50, 5, "State : "+req.ReceiverState, "0", 0, "L", false, 0, "")
	pdf.CellFormat(56, 5, "State Code : "+req.StateCode, "0", 1, "L", false, 0, "")

	pdf.Line(10, 75, 200, 75) // Horizontal line under Receiver details

	// Table Headers
	pdf.SetFont("Arial", "B", 9)
	pdf.SetXY(10, 75)
	
	// Print headers
	pdf.CellFormat(12, 8, "Sl.No.", "R", 0, "C", false, 0, "")
	pdf.CellFormat(85, 8, "Particulars", "R", 0, "C", false, 0, "")
	pdf.CellFormat(23, 8, "HSN Code", "R", 0, "C", false, 0, "")
	pdf.CellFormat(15, 8, "QTY", "R", 0, "C", false, 0, "")
	pdf.CellFormat(20, 8, "Rate", "R", 0, "C", false, 0, "")
	pdf.CellFormat(35, 8, "Amount", "", 1, "C", false, 0, "")
	
	// Double header border
	pdf.Line(10, 83, 200, 83)

	// Draw Vertical Lines of the Table
	// SlNo line: x=22, Particulars line: x=107, HSN line: x=130, QTY line: x=145, Rate line: x=165
	// The table runs from y=75 to y=190 (a height of 115mm)
	tableEndY := 190.0
	pdf.Line(22, 75, 22, tableEndY)
	pdf.Line(107, 75, 107, tableEndY)
	pdf.Line(130, 75, 130, tableEndY)
	pdf.Line(145, 75, 145, tableEndY)
	pdf.Line(165, 75, 165, tableEndY)
	// Amount Rs./Ps. split line: x=185 (from y=83 to y=190)
	pdf.Line(185, 83, 185, tableEndY)

	// Draw Rs. Ps. labels in Amount header area
	pdf.SetFont("Arial", "B", 7)
	pdf.SetXY(165, 80)
	pdf.CellFormat(20, 3, "Rs.", "R", 0, "C", false, 0, "")
	pdf.CellFormat(15, 3, "Ps.", "", 1, "C", false, 0, "")

	// Render Table Items
	pdf.SetFont("Arial", "", 9)
	yPos := 84.0
	for _, item := range req.Items {
		if yPos > 185 {
			break // avoid overflow
		}
		
		pdf.SetXY(10, yPos)
		pdf.CellFormat(12, 5, fmt.Sprintf("%d", item.SlNo), "", 0, "C", false, 0, "")
		
		// Align particulars to center under the header
		pdf.SetX(23)
		pdf.CellFormat(83, 5, item.Particulars, "", 0, "C", false, 0, "")
		
		pdf.SetX(107)
		pdf.CellFormat(23, 5, item.HSNCode, "", 0, "C", false, 0, "")
		
		pdf.SetX(130)
		pdf.CellFormat(15, 5, fmt.Sprintf("%.0f", item.QTY), "", 0, "C", false, 0, "")
		
		pdf.SetX(145)
		pdf.CellFormat(20, 5, fmt.Sprintf("%.2f", item.Rate), "", 0, "R", false, 0, "")
		
		// Split Amount into Rs. and Ps.
		rs, ps := splitAmount(item.Amount)
		pdf.SetX(165)
		pdf.CellFormat(20, 5, rs, "", 0, "R", false, 0, "")
		pdf.SetX(185)
		pdf.CellFormat(15, 5, ps, "", 1, "C", false, 0, "")
		
		yPos += 5
	}

	// Bottom line of the table
	pdf.Line(10, tableEndY, 200, tableEndY)

	// Summary / Calculations Block
	// Total Invoice Amount in words
	pdf.SetXY(11, 192)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(100, 4, "Total Invoice Amount (in words) ___________________________________________", "", 0, "L", false, 0, "")
	
	pdf.SetXY(11, 198)
	pdf.SetFont("Arial", "", 8)
	// We split words if they are long
	words := req.TotalAmountInWords
	if len(words) > 60 {
		pdf.CellFormat(100, 4, words[:60], "", 1, "L", false, 0, "")
		pdf.SetX(11)
		pdf.CellFormat(100, 4, words[60:], "", 0, "L", false, 0, "")
	} else {
		pdf.CellFormat(100, 4, words, "", 0, "L", false, 0, "")
	}

	// Dynamic Calculation Grid (Right side, from x=130 to x=200)
	pdf.SetFont("Arial", "B", 9)
	calcY := tableEndY // 190

	// Draw horizontal separators in calc block
	// Columns: Description (x=130 to x=165), Value Rs/Ps (x=165 to x=200, split at 185)
	
	drawCalcRow := func(label string, val float64, isBold bool) {
		pdf.SetXY(130, calcY)
		if isBold {
			pdf.SetFont("Arial", "B", 9)
		} else {
			pdf.SetFont("Arial", "", 9)
		}
		pdf.CellFormat(35, 6, label, "R", 0, "L", false, 0, "")
		rs, ps := splitAmount(val)
		pdf.CellFormat(20, 6, rs, "R", 0, "R", false, 0, "")
		pdf.CellFormat(15, 6, ps, "", 1, "C", false, 0, "")
		
		calcY += 6
		pdf.Line(130, calcY, 200, calcY)
	}

	drawCalcRow("TOTAL", subtotal, true)
	drawCalcRow(fmt.Sprintf("CGST  %.1f %%", req.CGSTRate), cgstVal, false)
	drawCalcRow(fmt.Sprintf("SGST  %.1f %%", req.SGSTRate), sgstVal, false)
	drawCalcRow(fmt.Sprintf("IGST  %.1f %%", req.IGSTRate), igstVal, false)
	drawCalcRow("Round Off", req.RoundOff, false)
	drawCalcRow("Total Invoice Value", finalValue, true)

	// Vertical line completing the calculation table
	pdf.Line(130, tableEndY, 130, calcY)
	pdf.Line(165, tableEndY, 165, calcY)
	pdf.Line(185, tableEndY, 185, calcY)

	// Bank Details (Permanent, Bottom Left)
	bankY := 228.0
	pdf.SetXY(12, bankY)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(100, 4, "Bank Name    : ICICI Bank", "", 1, "L", false, 0, "")
	pdf.SetX(12)
	pdf.CellFormat(100, 4, "A/C No.          : 79250550145", "", 1, "L", false, 0, "")
	pdf.SetX(12)
	pdf.CellFormat(100, 4, "IFSC No.        : ICIC0007925", "", 1, "L", false, 0, "")

	// Terms and conditions
	termsY := 243.0
	pdf.SetXY(12, termsY)
	pdf.SetFont("Arial", "BU", 8)
	pdf.CellFormat(100, 3, "Terms and conditions :", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 7)
	terms := []string{
		"1. Goods once sold cannot be taken back",
		"2. Interest chargeable @ 24% per annum if the bill not paid within 15 days",
		"3. Our responsibility ceases once the goods are handed to the carrier",
		"4. All disputes are subject to Sathupally Jurisdiction",
	}
	for _, term := range terms {
		pdf.SetX(12)
		pdf.CellFormat(100, 3.2, term, "", 1, "L", false, 0, "")
	}

	// Signature Area (Bottom Right)
	pdf.SetXY(110, 245)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(80, 4, "FOR : RAVINDRA SRI VENGAMAMBA EQUIPMENTS & RENTALS", "", 1, "R", false, 0, "")
	
	pdf.SetXY(110, 272)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(80, 4, "Authorised Signature", "", 0, "R", false, 0, "")

	// Save file
	return pdf.OutputFileAndClose(filePath)
}

func splitAmount(val float64) (string, string) {
	if val == 0 {
		return "0", "00"
	}
	// Round to 2 decimal places
	rounded := math.Round(val*100.0) / 100.0
	
	// Convert to string and split
	valStr := fmt.Sprintf("%.2f", rounded)
	parts := strings.Split(valStr, ".")
	
	rs := parts[0]
	ps := "00"
	if len(parts) > 1 {
		ps = parts[1]
	}
	
	// Add thousands separator if appropriate (optional, but clean)
	return rs, ps
}
