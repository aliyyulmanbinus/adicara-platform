// Code generated from PLANNER_SPEC (03 BE sections 6.3, 6.4 and 6.6) by a one-off script; edit freely afterwards,
// TestTemplatesMatchSpec keeps the two in step.

package domain

// samplePayments is the realisation of Wedding_Planner.pdf p. 5-6, laid over the 14 template lines of the sample project (03 BE 6.3).
// The PDF has no payment dates, so none is stored.
var samplePayments = []SamplePayment{
	{Scope: "lamaran", Key: "catering", Estimate: 2_000_000, DP: 500_000, Term1: 1_000_000, Term2: 500_000},
	{Scope: "lamaran", Key: "gaun_kebaya", Estimate: 750_000, DP: 750_000, Term1: 0, Term2: 0},
	{Scope: "lamaran", Key: "fotografer", Estimate: 1_000_000, DP: 500_000, Term1: 0, Term2: 0},
	{Scope: "lamaran", Key: "cincin", Estimate: 2_500_000, DP: 600_000, Term1: 0, Term2: 0},
	{Scope: "lamaran", Key: "dekorasi", Estimate: 900_000, DP: 900_000, Term1: 0, Term2: 0},
	{Scope: "akad", Key: "biaya_kua", Estimate: 600_000, DP: 600_000, Term1: 0, Term2: 0},
	{Scope: "resepsi", Key: "mahar", Estimate: 3_000_000, DP: 3_000_000, Term1: 0, Term2: 0},
	{Scope: "resepsi", Key: "catering", Estimate: 25_000_000, DP: 5_000_000, Term1: 10_000_000, Term2: 0},
	{Scope: "resepsi", Key: "rias_busana", Estimate: 5_000_000, DP: 600_000, Term1: 0, Term2: 0},
	{Scope: "resepsi", Key: "venue_dekor", Estimate: 20_000_000, DP: 900_000, Term1: 0, Term2: 0},
	{Scope: "resepsi", Key: "undangan_souvenir", Estimate: 3_500_000, DP: 1_500_000, Term1: 0, Term2: 0},
	{Scope: "resepsi", Key: "dokumentasi", Estimate: 3_000_000, DP: 300_000, Term1: 0, Term2: 0},
	{Scope: "resepsi", Key: "seserahan", Estimate: 3_500_000, DP: 3_500_000, Term1: 0, Term2: 0},
	{Scope: "resepsi", Key: "hiburan", Estimate: 3_400_000, DP: 1_400_000, Term1: 1_000_000, Term2: 1_000_000},
}

// sampleSavings are the monthly deposits of Wedding_Planner.pdf p. 7 (03 BE 6.4): February to December 2026, on the 1st of the month.
var sampleSavings = []sampleMonth{
	{Month: 2, Groom: 2_500_000, Bride: 500_000},
	{Month: 3, Groom: 5_000_000, Bride: 2_000_000},
	{Month: 4, Groom: 1_500_000, Bride: 1_000_000},
	{Month: 5, Groom: 3_000_000, Bride: 500_000},
	{Month: 6, Groom: 1_500_000, Bride: 1_000_000},
	{Month: 7, Groom: 2_500_000, Bride: 1_500_000},
	{Month: 8, Groom: 2_500_000, Bride: 1_500_000},
	{Month: 9, Groom: 1_500_000, Bride: 2_500_000},
	{Month: 10, Groom: 2_000_000, Bride: 1_000_000},
	{Month: 11, Groom: 2_000_000, Bride: 2_000_000},
	{Month: 12, Groom: 6_000_000, Bride: 4_000_000},
}
