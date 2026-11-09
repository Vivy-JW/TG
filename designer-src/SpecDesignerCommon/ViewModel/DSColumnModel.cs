using System;
using System.Xml.Linq;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000094 RID: 148
	public class DSColumnModel
	{
		// Token: 0x170001B7 RID: 439
		// (get) Token: 0x06000602 RID: 1538 RVA: 0x0001B8A6 File Offset: 0x00019AA6
		// (set) Token: 0x06000603 RID: 1539 RVA: 0x0001B8AE File Offset: 0x00019AAE
		public string TBLName { get; set; }

		// Token: 0x170001B8 RID: 440
		// (get) Token: 0x06000604 RID: 1540 RVA: 0x0001B8B7 File Offset: 0x00019AB7
		// (set) Token: 0x06000605 RID: 1541 RVA: 0x0001B8BF File Offset: 0x00019ABF
		public string Name { get; set; }

		// Token: 0x170001B9 RID: 441
		// (get) Token: 0x06000606 RID: 1542 RVA: 0x0001B8C8 File Offset: 0x00019AC8
		// (set) Token: 0x06000607 RID: 1543 RVA: 0x0001B8D0 File Offset: 0x00019AD0
		public XElement Detail { get; set; }

		// Token: 0x170001BA RID: 442
		// (get) Token: 0x06000608 RID: 1544 RVA: 0x0001B8D9 File Offset: 0x00019AD9
		// (set) Token: 0x06000609 RID: 1545 RVA: 0x0001B8E1 File Offset: 0x00019AE1
		public string Description { get; set; }

		// Token: 0x170001BB RID: 443
		// (get) Token: 0x0600060A RID: 1546 RVA: 0x0001B8EA File Offset: 0x00019AEA
		// (set) Token: 0x0600060B RID: 1547 RVA: 0x0001B8F2 File Offset: 0x00019AF2
		public bool IsUsed { get; set; }
	}
}
