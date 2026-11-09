using System;
using System.Xml.Linq;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000048 RID: 72
	public class ADPModel
	{
		// Token: 0x0600022E RID: 558 RVA: 0x0000B072 File Offset: 0x00009272
		public ADPModel()
		{
			this.IsDiff = false;
			this.Type = "ADP";
		}

		// Token: 0x0600022F RID: 559 RVA: 0x0000B08C File Offset: 0x0000928C
		public ADPModel(string name)
			: this()
		{
			this.Name = name;
		}

		// Token: 0x170000BF RID: 191
		// (get) Token: 0x06000230 RID: 560 RVA: 0x0000B09B File Offset: 0x0000929B
		// (set) Token: 0x06000231 RID: 561 RVA: 0x0000B0A3 File Offset: 0x000092A3
		public XElement Element { get; set; }

		// Token: 0x170000C0 RID: 192
		// (get) Token: 0x06000232 RID: 562 RVA: 0x0000B0AC File Offset: 0x000092AC
		// (set) Token: 0x06000233 RID: 563 RVA: 0x0000B0B4 File Offset: 0x000092B4
		public string Name { get; private set; }

		// Token: 0x170000C1 RID: 193
		// (get) Token: 0x06000234 RID: 564 RVA: 0x0000B0BD File Offset: 0x000092BD
		// (set) Token: 0x06000235 RID: 565 RVA: 0x0000B0C5 File Offset: 0x000092C5
		public bool IsDiff { get; set; }

		// Token: 0x170000C2 RID: 194
		// (get) Token: 0x06000236 RID: 566 RVA: 0x0000B0CE File Offset: 0x000092CE
		// (set) Token: 0x06000237 RID: 567 RVA: 0x0000B0D6 File Offset: 0x000092D6
		public bool CusToStd { get; set; }

		// Token: 0x170000C3 RID: 195
		// (get) Token: 0x06000238 RID: 568 RVA: 0x0000B0DF File Offset: 0x000092DF
		// (set) Token: 0x06000239 RID: 569 RVA: 0x0000B0E7 File Offset: 0x000092E7
		public string Type { get; set; }
	}
}
