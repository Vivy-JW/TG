using System;
using System.Collections.ObjectModel;

namespace SpecDesignerCommon.Bookmark
{
	// Token: 0x0200007A RID: 122
	public interface IBookmarkMargin
	{
		// Token: 0x1700013E RID: 318
		// (get) Token: 0x0600049D RID: 1181
		ObservableCollection<IBookmark> Bookmarks { get; }

		// Token: 0x0600049E RID: 1182
		void Redraw();

		// Token: 0x14000012 RID: 18
		// (add) Token: 0x0600049F RID: 1183
		// (remove) Token: 0x060004A0 RID: 1184
		event EventHandler RedrawRequested;

		// Token: 0x14000013 RID: 19
		// (add) Token: 0x060004A1 RID: 1185
		// (remove) Token: 0x060004A2 RID: 1186
		event EventHandler SelectedBookmarkChanged;

		// Token: 0x060004A3 RID: 1187
		int GetNextIndex();

		// Token: 0x1700013F RID: 319
		// (get) Token: 0x060004A4 RID: 1188
		// (set) Token: 0x060004A5 RID: 1189
		IBookmark SelectedBookmark { get; set; }
	}
}
