using System;

namespace SpecDesignerCommon
{
	// Token: 0x0200003C RID: 60
	public class ReplaceResultInfo
	{
		// Token: 0x060001E8 RID: 488 RVA: 0x00008CE6 File Offset: 0x00006EE6
		public ReplaceResultInfo(SearchResultInfo args)
		{
			this.SearchInformation = args;
		}

		// Token: 0x060001E9 RID: 489 RVA: 0x00008CF5 File Offset: 0x00006EF5
		public ReplaceResultInfo()
		{
		}

		// Token: 0x17000071 RID: 113
		// (get) Token: 0x060001EA RID: 490 RVA: 0x00008CFD File Offset: 0x00006EFD
		// (set) Token: 0x060001EB RID: 491 RVA: 0x00008D05 File Offset: 0x00006F05
		public string ReplaceAs { get; set; }

		// Token: 0x17000072 RID: 114
		// (get) Token: 0x060001EC RID: 492 RVA: 0x00008D0E File Offset: 0x00006F0E
		// (set) Token: 0x060001ED RID: 493 RVA: 0x00008D16 File Offset: 0x00006F16
		public SearchResultInfo SearchInformation { get; set; }

		// Token: 0x17000073 RID: 115
		// (get) Token: 0x060001EE RID: 494 RVA: 0x00008D1F File Offset: 0x00006F1F
		// (set) Token: 0x060001EF RID: 495 RVA: 0x00008D27 File Offset: 0x00006F27
		public int startLine { get; set; }

		// Token: 0x17000074 RID: 116
		// (get) Token: 0x060001F0 RID: 496 RVA: 0x00008D30 File Offset: 0x00006F30
		// (set) Token: 0x060001F1 RID: 497 RVA: 0x00008D38 File Offset: 0x00006F38
		public int endLine { get; set; }

		// Token: 0x17000075 RID: 117
		// (get) Token: 0x060001F2 RID: 498 RVA: 0x00008D41 File Offset: 0x00006F41
		// (set) Token: 0x060001F3 RID: 499 RVA: 0x00008D49 File Offset: 0x00006F49
		public bool IsMatchCase { get; set; }

		// Token: 0x17000076 RID: 118
		// (get) Token: 0x060001F4 RID: 500 RVA: 0x00008D52 File Offset: 0x00006F52
		// (set) Token: 0x060001F5 RID: 501 RVA: 0x00008D5A File Offset: 0x00006F5A
		public bool SelectedOnly { get; set; }

		// Token: 0x17000077 RID: 119
		// (get) Token: 0x060001F6 RID: 502 RVA: 0x00008D63 File Offset: 0x00006F63
		// (set) Token: 0x060001F7 RID: 503 RVA: 0x00008D6B File Offset: 0x00006F6B
		public string Keyword { get; set; }
	}
}
