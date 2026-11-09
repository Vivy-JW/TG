using System;
using System.Windows;

namespace SpecDesigner.Infrastructure.Extension
{
	// Token: 0x02000039 RID: 57
	public interface IDropable
	{
		// Token: 0x17000074 RID: 116
		// (get) Token: 0x06000158 RID: 344
		Type AllowType { get; }

		// Token: 0x06000159 RID: 345
		void DropOver(DragEventArgs dragEvetnArgs);

		// Token: 0x0600015A RID: 346
		void Drop(IDragable drag);

		// Token: 0x17000075 RID: 117
		// (get) Token: 0x0600015B RID: 347
		bool CanDrop { get; }

		// Token: 0x0600015C RID: 348
		bool CheckDropable(IDragable dragable);
	}
}
