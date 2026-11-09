using System;
using System.Windows;

namespace SpecDesignerCommon.Site.Behaviors
{
	// Token: 0x02000034 RID: 52
	public interface IDropable
	{
		// Token: 0x1700006B RID: 107
		// (get) Token: 0x060001B8 RID: 440
		Type AllowType { get; }

		// Token: 0x060001B9 RID: 441
		void DropOver(DragEventArgs dragEvetnArgs);

		// Token: 0x060001BA RID: 442
		void Drop(IDragable drag);

		// Token: 0x060001BB RID: 443
		bool CanDrop(IDragable drag);
	}
}
