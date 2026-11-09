using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x02000128 RID: 296
	public class ReplaceAllKeywordEventArgs
	{
		// Token: 0x170002B5 RID: 693
		// (get) Token: 0x06000A5E RID: 2654 RVA: 0x00033A68 File Offset: 0x00031C68
		// (set) Token: 0x06000A5F RID: 2655 RVA: 0x00033A70 File Offset: 0x00031C70
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x170002B6 RID: 694
		// (get) Token: 0x06000A60 RID: 2656 RVA: 0x00033A79 File Offset: 0x00031C79
		// (set) Token: 0x06000A61 RID: 2657 RVA: 0x00033A81 File Offset: 0x00031C81
		public string Keyword { get; private set; }

		// Token: 0x170002B7 RID: 695
		// (get) Token: 0x06000A62 RID: 2658 RVA: 0x00033A8A File Offset: 0x00031C8A
		// (set) Token: 0x06000A63 RID: 2659 RVA: 0x00033A92 File Offset: 0x00031C92
		public bool IsMatchCase { get; private set; }

		// Token: 0x170002B8 RID: 696
		// (get) Token: 0x06000A64 RID: 2660 RVA: 0x00033A9B File Offset: 0x00031C9B
		// (set) Token: 0x06000A65 RID: 2661 RVA: 0x00033AA3 File Offset: 0x00031CA3
		public bool IsRegExMode { get; set; }

		// Token: 0x170002B9 RID: 697
		// (get) Token: 0x06000A66 RID: 2662 RVA: 0x00033AAC File Offset: 0x00031CAC
		// (set) Token: 0x06000A67 RID: 2663 RVA: 0x00033AB4 File Offset: 0x00031CB4
		public bool EditableAreaOnly { get; private set; }

		// Token: 0x170002BA RID: 698
		// (get) Token: 0x06000A68 RID: 2664 RVA: 0x00033ABD File Offset: 0x00031CBD
		// (set) Token: 0x06000A69 RID: 2665 RVA: 0x00033AC5 File Offset: 0x00031CC5
		public string ReplaceAs { get; set; }

		// Token: 0x170002BB RID: 699
		// (get) Token: 0x06000A6A RID: 2666 RVA: 0x00033ACE File Offset: 0x00031CCE
		// (set) Token: 0x06000A6B RID: 2667 RVA: 0x00033AD6 File Offset: 0x00031CD6
		public bool SelectedOnly { get; private set; }

		// Token: 0x06000A6C RID: 2668 RVA: 0x00033ADF File Offset: 0x00031CDF
		public ReplaceAllKeywordEventArgs(PackageKey key, string keyword, bool isMatchCase, bool editableAreaOnly, bool selectedOnly, bool isRegExMode)
		{
			this.ProgramKey = key;
			this.Keyword = keyword;
			this.IsMatchCase = isMatchCase;
			this.EditableAreaOnly = editableAreaOnly;
		}
	}
}
