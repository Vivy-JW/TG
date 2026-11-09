using System;
using System.Windows.Input;

namespace SpecDesignerCommon.Bookmark
{
	// Token: 0x0200007B RID: 123
	public interface IBookmark
	{
		// Token: 0x17000140 RID: 320
		// (get) Token: 0x060004A6 RID: 1190
		string Name { get; }

		// Token: 0x17000141 RID: 321
		// (get) Token: 0x060004A7 RID: 1191
		int LineNumber { get; }

		// Token: 0x17000142 RID: 322
		// (get) Token: 0x060004A8 RID: 1192
		int ZOrder { get; }

		// Token: 0x060004A9 RID: 1193
		void MouseDown(MouseButtonEventArgs e);

		// Token: 0x060004AA RID: 1194
		void MouseUp(MouseButtonEventArgs e);
	}
}
