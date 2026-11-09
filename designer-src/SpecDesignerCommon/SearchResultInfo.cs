using System;

namespace SpecDesignerCommon
{
	// Token: 0x02000146 RID: 326
	public class SearchResultInfo
	{
		// Token: 0x06000B78 RID: 2936 RVA: 0x00038BAF File Offset: 0x00036DAF
		public SearchResultInfo()
		{
			this.IsEditable = true;
			this.Memo = -1;
			this.Mode = SearchMode.Regular;
			this.Direction = SearchDirection.Forward;
		}

		// Token: 0x170002F3 RID: 755
		// (get) Token: 0x06000B79 RID: 2937 RVA: 0x00038BD3 File Offset: 0x00036DD3
		// (set) Token: 0x06000B7A RID: 2938 RVA: 0x00038BDB File Offset: 0x00036DDB
		public PackageKey ProgramKey { get; set; }

		// Token: 0x170002F4 RID: 756
		// (get) Token: 0x06000B7B RID: 2939 RVA: 0x00038BE4 File Offset: 0x00036DE4
		// (set) Token: 0x06000B7C RID: 2940 RVA: 0x00038BEC File Offset: 0x00036DEC
		public TzpType SourceType { get; set; }

		// Token: 0x170002F5 RID: 757
		// (get) Token: 0x06000B7D RID: 2941 RVA: 0x00038BF5 File Offset: 0x00036DF5
		// (set) Token: 0x06000B7E RID: 2942 RVA: 0x00038BFD File Offset: 0x00036DFD
		public string Keyword { get; set; }

		// Token: 0x170002F6 RID: 758
		// (get) Token: 0x06000B7F RID: 2943 RVA: 0x00038C06 File Offset: 0x00036E06
		// (set) Token: 0x06000B80 RID: 2944 RVA: 0x00038C0E File Offset: 0x00036E0E
		public string Match { get; set; }

		// Token: 0x170002F7 RID: 759
		// (get) Token: 0x06000B81 RID: 2945 RVA: 0x00038C17 File Offset: 0x00036E17
		// (set) Token: 0x06000B82 RID: 2946 RVA: 0x00038C1F File Offset: 0x00036E1F
		public string Key { get; set; }

		// Token: 0x170002F8 RID: 760
		// (get) Token: 0x06000B83 RID: 2947 RVA: 0x00038C28 File Offset: 0x00036E28
		// (set) Token: 0x06000B84 RID: 2948 RVA: 0x00038C30 File Offset: 0x00036E30
		public bool IsEditable { get; set; }

		// Token: 0x170002F9 RID: 761
		// (get) Token: 0x06000B85 RID: 2949 RVA: 0x00038C39 File Offset: 0x00036E39
		// (set) Token: 0x06000B86 RID: 2950 RVA: 0x00038C41 File Offset: 0x00036E41
		public int Memo { get; set; }

		// Token: 0x170002FA RID: 762
		// (get) Token: 0x06000B87 RID: 2951 RVA: 0x00038C4A File Offset: 0x00036E4A
		// (set) Token: 0x06000B88 RID: 2952 RVA: 0x00038C52 File Offset: 0x00036E52
		public SearchMode Mode { get; set; }

		// Token: 0x170002FB RID: 763
		// (get) Token: 0x06000B89 RID: 2953 RVA: 0x00038C5B File Offset: 0x00036E5B
		// (set) Token: 0x06000B8A RID: 2954 RVA: 0x00038C63 File Offset: 0x00036E63
		public SearchDirection Direction { get; set; }

		// Token: 0x170002FC RID: 764
		// (get) Token: 0x06000B8B RID: 2955 RVA: 0x00038C6C File Offset: 0x00036E6C
		// (set) Token: 0x06000B8C RID: 2956 RVA: 0x00038C74 File Offset: 0x00036E74
		public bool ComparingSame { get; set; }
	}
}
