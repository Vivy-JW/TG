using System;
using System.Xml.Linq;

namespace SpecDesignerCommon
{
	// Token: 0x0200011B RID: 283
	public class SpecPropertiesEventArgs
	{
		// Token: 0x170002A2 RID: 674
		// (get) Token: 0x06000A0C RID: 2572 RVA: 0x000320F2 File Offset: 0x000302F2
		// (set) Token: 0x06000A0D RID: 2573 RVA: 0x000320FA File Offset: 0x000302FA
		public string OldName { get; set; }

		// Token: 0x170002A3 RID: 675
		// (get) Token: 0x06000A0E RID: 2574 RVA: 0x00032103 File Offset: 0x00030303
		// (set) Token: 0x06000A0F RID: 2575 RVA: 0x0003210B File Offset: 0x0003030B
		public XElement Attributes { get; set; }

		// Token: 0x06000A10 RID: 2576 RVA: 0x00032114 File Offset: 0x00030314
		public SpecPropertiesEventArgs(string oldName, XElement attr)
		{
			this.OldName = oldName;
			this.Attributes = attr;
		}
	}
}
