using System;
using System.Windows.Input;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000011 RID: 17
	internal static class TabIndexCommands
	{
		// Token: 0x04000039 RID: 57
		public static RoutedCommand SetAsFirstCommand = new RoutedCommand();

		// Token: 0x0400003A RID: 58
		public static RoutedCommand SetAsNextcommand = new RoutedCommand();

		// Token: 0x0400003B RID: 59
		public static RoutedCommand SetAsCurrentCommand = new RoutedCommand();

		// Token: 0x0400003C RID: 60
		public static RoutedCommand ShiftCurrentCommand = new RoutedCommand();

		// Token: 0x0400003D RID: 61
		public static RoutedCommand SwapSelectedCommand = new RoutedCommand();

		// Token: 0x0400003E RID: 62
		public static RoutedCommand SetAsNonTabableCommand = new RoutedCommand();
	}
}
