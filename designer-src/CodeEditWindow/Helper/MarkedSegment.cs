using System;
using ICSharpCode.AvalonEdit.Document;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000022 RID: 34
	public class MarkedSegment
	{
		// Token: 0x17000024 RID: 36
		// (get) Token: 0x0600013E RID: 318 RVA: 0x0000C245 File Offset: 0x0000A445
		// (set) Token: 0x0600013F RID: 319 RVA: 0x0000C24D File Offset: 0x0000A44D
		public DocumentLine StartLine { get; set; }

		// Token: 0x17000025 RID: 37
		// (get) Token: 0x06000140 RID: 320 RVA: 0x0000C256 File Offset: 0x0000A456
		// (set) Token: 0x06000141 RID: 321 RVA: 0x0000C25E File Offset: 0x0000A45E
		public DocumentLine EndLine { get; set; }
	}
}
