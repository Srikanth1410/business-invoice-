import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';

interface InvoiceItem {
  slNo: number;
  particulars: string;
  hsnCode: string;
  qty: number;
  rate: number;
  amount: number;
}

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App implements OnInit {
  title = 'Invoice Generator';
  Math = Math;

  // Invoice fields
  invoiceNo: string = '';
  invoiceDate: string = '';
  receiverName: string = '';
  receiverAddress: string = '';
  receiverGstin: string = '';
  receiverState: string = 'Telangana';
  stateCode: string = '36';

  items: InvoiceItem[] = [
    { slNo: 1, particulars: '', hsnCode: '', qty: 0, rate: 0, amount: 0 }
  ];

  cgstRate: number = 9;
  sgstRate: number = 9;
  igstRate: number = 0;
  
  // Computed values
  rawSubtotal: number = 0;
  cgstAmount: number = 0;
  sgstAmount: number = 0;
  igstAmount: number = 0;
  roundOff: number = 0;
  totalInvoiceValue: number = 0;
  totalAmountInWords: string = '';

  // API interaction status
  loading: boolean = false;
  success: boolean = false;
  message: string = '';
  savedFilePath: string = '';

  // Constant Bank details (for display, but pre-filled in preview)
  bankDetails = {
    name: 'ICICI Bank',
    accountNo: '79250550145',
    ifsc: 'ICIC0007925'
  };

  constructor(private http: HttpClient) {}

  ngOnInit() {
    // Pre-fill today's date
    const today = new Date();
    const dd = String(today.getDate()).padStart(2, '0');
    const mm = String(today.getMonth() + 1).padStart(2, '0'); // January is 0!
    const yyyy = today.getFullYear();
    this.invoiceDate = `${yyyy}-${mm}-${dd}`;
    this.calculateTotals();
  }

  addItem() {
    const nextSlNo = this.items.length + 1;
    this.items.push({
      slNo: nextSlNo,
      particulars: '',
      hsnCode: '',
      qty: 0,
      rate: 0,
      amount: 0
    });
    this.calculateTotals();
  }

  removeItem(index: number) {
    if (this.items.length > 1) {
      this.items.splice(index, 1);
      // Re-index Sl No.
      this.items.forEach((item, idx) => {
        item.slNo = idx + 1;
      });
      this.calculateTotals();
    }
  }

  calculateTotals() {
    let subtotal = 0;
    this.items.forEach(item => {
      item.amount = (item.qty || 0) * (item.rate || 0);
      subtotal += item.amount;
    });

    this.rawSubtotal = subtotal;

    // Calculate taxes with 2 decimal rounding
    this.cgstAmount = Math.round((subtotal * (this.cgstRate / 100)) * 100) / 100;
    this.sgstAmount = Math.round((subtotal * (this.sgstRate / 100)) * 100) / 100;
    this.igstAmount = Math.round((subtotal * (this.igstRate / 100)) * 100) / 100;

    const preciseTotal = subtotal + this.cgstAmount + this.sgstAmount + this.igstAmount;
    const roundedTotal = Math.round(preciseTotal);
    
    // Round Off is difference between rounded and precise total
    this.roundOff = Math.round((roundedTotal - preciseTotal) * 100) / 100;
    this.totalInvoiceValue = roundedTotal;

    this.totalAmountInWords = numberToWords(this.totalInvoiceValue);
  }

  onSubmit() {
    this.calculateTotals(); // Double check calculations
    
    const payload = {
      invoiceNo: this.invoiceNo,
      invoiceDate: this.invoiceDate,
      receiverName: this.receiverName,
      receiverAddress: this.receiverAddress,
      receiverGstin: this.receiverGstin,
      receiverState: this.receiverState,
      stateCode: this.stateCode,
      items: this.items,
      cgstRate: this.cgstRate,
      sgstRate: this.sgstRate,
      igstRate: this.igstRate,
      roundOff: this.roundOff,
      totalInvoiceValue: this.totalInvoiceValue,
      totalAmountInWords: this.totalAmountInWords
    };

    if (!this.invoiceNo || !this.receiverName) {
      this.message = 'Please enter Invoice Number and Customer Name!';
      this.success = false;
      return;
    }

    this.loading = true;
    this.message = '';
    this.savedFilePath = '';

    this.http.post<any>('http://localhost:8080/api/invoice', payload)
      .subscribe({
        next: (res) => {
          this.loading = false;
          if (res.success) {
            this.success = true;
            this.message = res.message;
            this.savedFilePath = res.filePath;
          } else {
            this.success = false;
            this.message = res.message || 'Failed to generate invoice.';
          }
        },
        error: (err) => {
          this.loading = false;
          this.success = false;
          this.message = 'Error connecting to Go backend server. Make sure the Go API server is running on port 8080.';
          console.error(err);
        }
      });
  }

  closeModal() {
    this.success = false;
    this.message = '';
    this.savedFilePath = '';
  }

  onPrint() {
    window.print();
  }
}

// Number to Indian Currency Words Helper Function
function numberToWords(num: number): string {
  if (num === 0) return 'Zero Rupees Only';
  const a = ['', 'One ', 'Two ', 'Three ', 'Four ', 'Five ', 'Six ', 'Seven ', 'Eight ', 'Nine ', 'Ten ', 'Eleven ', 'Twelve ', 'Thirteen ', 'Fourteen ', 'Fifteen ', 'Sixteen ', 'Seventeen ', 'Eighteen ', 'Nineteen '];
  const b = ['', '', 'Twenty', 'Thirty', 'Forty', 'Fifty', 'Sixty', 'Seventy', 'Eighty', 'Ninety'];
  
  function numToWords(n: number): string {
    if (n < 20) return a[n];
    if (n < 100) return b[Math.floor(n / 10)] + (n % 10 !== 0 ? ' ' + a[n % 10] : '');
    if (n < 1000) return a[Math.floor(n / 100)] + 'Hundred ' + (n % 100 !== 0 ? 'and ' + numToWords(n % 100) : '');
    return '';
  }

  const rupees = Math.floor(num);
  const paise = Math.round((num - rupees) * 100);
  
  let rupeesStr = '';
  if (rupees > 0) {
    if (rupees < 1000) {
      rupeesStr = numToWords(rupees);
    } else {
      const crore = Math.floor(rupees / 10000000);
      let remainder = rupees % 10000000;
      const lakh = Math.floor(remainder / 100000);
      remainder = remainder % 100000;
      const thousand = Math.floor(remainder / 1000);
      remainder = remainder % 1000;
      
      let str = '';
      if (crore > 0) str += numToWords(crore) + 'Crore ';
      if (lakh > 0) str += numToWords(lakh) + 'Lakh ';
      if (thousand > 0) str += numToWords(thousand) + 'Thousand ';
      if (remainder > 0) str += numToWords(remainder);
      rupeesStr = str;
    }
    rupeesStr += 'Rupees ';
  }

  let paiseStr = '';
  if (paise > 0) {
    paiseStr = (rupeesStr !== '' ? 'and ' : '') + numToWords(paise) + 'Paise ';
  }
  
  return rupeesStr + paiseStr + 'Only';
}
