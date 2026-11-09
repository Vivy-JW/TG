using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x02000069 RID: 105
	public class DocumentErrorsEventArgs
	{
		// Token: 0x17000109 RID: 265
		// (get) Token: 0x060003FD RID: 1021 RVA: 0x00012ABF File Offset: 0x00010CBF
		// (set) Token: 0x060003FE RID: 1022 RVA: 0x00012AC7 File Offset: 0x00010CC7
		public PackageKey ProgramKey { get; set; }

		// Token: 0x1700010A RID: 266
		// (get) Token: 0x060003FF RID: 1023 RVA: 0x00012AD0 File Offset: 0x00010CD0
		// (set) Token: 0x06000400 RID: 1024 RVA: 0x00012AD8 File Offset: 0x00010CD8
		public DateTime Time { get; set; }

		// Token: 0x1700010B RID: 267
		// (get) Token: 0x06000401 RID: 1025 RVA: 0x00012AE1 File Offset: 0x00010CE1
		// (set) Token: 0x06000402 RID: 1026 RVA: 0x00012AE9 File Offset: 0x00010CE9
		public string Description { get; set; }

		// Token: 0x1700010C RID: 268
		// (get) Token: 0x06000403 RID: 1027 RVA: 0x00012AF2 File Offset: 0x00010CF2
		// (set) Token: 0x06000404 RID: 1028 RVA: 0x00012AFA File Offset: 0x00010CFA
		public TzpType SourceType { get; set; }

		// Token: 0x1700010D RID: 269
		// (get) Token: 0x06000405 RID: 1029 RVA: 0x00012B03 File Offset: 0x00010D03
		// (set) Token: 0x06000406 RID: 1030 RVA: 0x00012B0B File Offset: 0x00010D0B
		public string Key { get; set; }

		// Token: 0x1700010E RID: 270
		// (get) Token: 0x06000407 RID: 1031 RVA: 0x00012B14 File Offset: 0x00010D14
		// (set) Token: 0x06000408 RID: 1032 RVA: 0x00012B1C File Offset: 0x00010D1C
		public ErrorsType ErrorType { get; set; }
	}
}
