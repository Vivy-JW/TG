using System;
using System.Windows.Input;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x02000036 RID: 54
	public class BookmarkCommands
	{
		// Token: 0x0400008E RID: 142
		public static RoutedCommand ToggleBookmarkCommand = new RoutedCommand("ToggleBookmarkCommand", typeof(BookmarkCommands));

		// Token: 0x0400008F RID: 143
		public static RoutedCommand ClearBookmarkCommand = new RoutedCommand("ClearBookmarkCommand", typeof(BookmarkCommands));

		// Token: 0x04000090 RID: 144
		public static RoutedCommand PreviousBookmarkCommand = new RoutedCommand("PreviousBookmarkCommand", typeof(BookmarkCommands));

		// Token: 0x04000091 RID: 145
		public static RoutedCommand NextBookmarkCommand = new RoutedCommand("NextBookmarkCommand", typeof(BookmarkCommands));

		// Token: 0x04000092 RID: 146
		public static RoutedCommand DeleteBookmarkCommand = new RoutedCommand("DeleteBookmarkCommand", typeof(BookmarkCommands));

		// Token: 0x04000093 RID: 147
		public static RoutedCommand SelectBookmarkCommand = new RoutedCommand("SelectBookmarkCommand", typeof(BookmarkCommands));
	}
}
