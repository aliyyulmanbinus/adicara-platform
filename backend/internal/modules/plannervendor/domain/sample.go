package domain

// The vendor dataset of the sample project (04 BE 5.2). The PDF (p. 12) only lists categories with dummy rows, so the
// names, the phone numbers (pattern 08120000xxxx) and the prices are fictional; the PIC is left empty rather than invented.
// Seven vendors are fix, like the seven categories of the PDF; the other three are still being negotiated.

// SampleScope is the session of every sample vendor.
const SampleScope = "resepsi"

// SampleVendor is one vendor of the sample project; its pagu is not stored (it is read from the active scenario).
type SampleVendor struct {
	CategoryKey string
	Name        string
	Phone       string
	Price       int64
	Status      VendorStatus
}

var SampleVendors = []SampleVendor{
	{"venue", "Sasana Kriya Ballroom", "081200000001", 12_000_000, VendorFixed},
	{"dekorasi", "Pelaminan Mega Indah", "081200000002", 8_000_000, VendorFixed},
	{"wo", "Harmony Wedding Organizer", "081200000003", 5_000_000, VendorFixed},
	{"mua", "Sanggar Rias Ayu", "081200000004", 3_500_000, VendorFixed},
	{"catering", "Katering Bu Minah (500 pax)", "081200000005", 25_000_000, VendorFixed},
	{"fotografi", "Studio Cahaya", "081200000006", 3_000_000, VendorFixed},
	{"attire", "Butik Kebaya Anggun", "081200000007", 1_500_000, VendorFixed},
	{"hand_bouquet", "Fleur Hand Bouquet", "081200000008", 900_000, VendorPlanned},
	{"mc", "Kang Dedi Master MC", "081200000009", 2_000_000, VendorNegotiating},
	{"entertainment", "Akustik Nada Kirana", "081200000010", 1_400_000, VendorNegotiating},
}

// SampleLineLinks are the budget lines of the wedding template that the sample overlay points at a vendor (their estimates
// 25.000.000 and 3.000.000 already equal the deals, so the PDF totals stay whole). The overlay does not run lock-deal:
// Wedding Organizer would add a 15th line and change the total of the PDF.
var SampleLineLinks = []struct{ Scope, Key, VendorCategory string }{
	{"resepsi", "catering", "catering"},
	{"resepsi", "dokumentasi", "fotografi"},
}

// SampleTaskLink is the one task of the sample timeline that points at a vendor (02 BE 4.6).
var SampleTaskLink = struct{ TemplateKey, VendorCategory string }{"wedding.visit_venue_pernikahan", "venue"}
