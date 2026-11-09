using System;
using System.Windows;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200004C RID: 76
	public class ActionStatus : DependencyObject
	{
		// Token: 0x17000031 RID: 49
		// (get) Token: 0x060001D3 RID: 467 RVA: 0x0000C85B File Offset: 0x0000AA5B
		// (set) Token: 0x060001D4 RID: 468 RVA: 0x0000C863 File Offset: 0x0000AA63
		public string Status { get; set; }

		// Token: 0x17000032 RID: 50
		// (get) Token: 0x060001D5 RID: 469 RVA: 0x0000C86C File Offset: 0x0000AA6C
		// (set) Token: 0x060001D6 RID: 470 RVA: 0x0000C874 File Offset: 0x0000AA74
		public string Text { get; set; }
	}
}
