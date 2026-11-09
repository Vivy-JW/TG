using System;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x0200002F RID: 47
	public class CodeSamepleModel
	{
		// Token: 0x17000069 RID: 105
		// (get) Token: 0x0600013F RID: 319 RVA: 0x00006A5E File Offset: 0x00004C5E
		// (set) Token: 0x06000140 RID: 320 RVA: 0x00006A66 File Offset: 0x00004C66
		public string ID { get; set; }

		// Token: 0x1700006A RID: 106
		// (get) Token: 0x06000141 RID: 321 RVA: 0x00006A6F File Offset: 0x00004C6F
		// (set) Token: 0x06000142 RID: 322 RVA: 0x00006A77 File Offset: 0x00004C77
		public string Description { get; set; }

		// Token: 0x1700006B RID: 107
		// (get) Token: 0x06000143 RID: 323 RVA: 0x00006A80 File Offset: 0x00004C80
		// (set) Token: 0x06000144 RID: 324 RVA: 0x00006A88 File Offset: 0x00004C88
		public RangeEnum Range { get; set; }

		// Token: 0x1700006C RID: 108
		// (get) Token: 0x06000145 RID: 325 RVA: 0x00006A91 File Offset: 0x00004C91
		// (set) Token: 0x06000146 RID: 326 RVA: 0x00006A99 File Offset: 0x00004C99
		public bool IsShow { get; set; }

		// Token: 0x1700006D RID: 109
		// (get) Token: 0x06000147 RID: 327 RVA: 0x00006AA2 File Offset: 0x00004CA2
		// (set) Token: 0x06000148 RID: 328 RVA: 0x00006AAA File Offset: 0x00004CAA
		public string Content { get; set; }
	}
}
