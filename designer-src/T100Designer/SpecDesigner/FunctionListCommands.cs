using System;
using System.Windows.Input;

namespace SpecDesigner
{
	// Token: 0x0200000F RID: 15
	public static class FunctionListCommands
	{
		// Token: 0x0400003B RID: 59
		public static RoutedCommand SearchCommand = new RoutedCommand("SearchCommand", typeof(FunctionListCommands));

		// Token: 0x0400003C RID: 60
		public static RoutedCommand AddCommand = new RoutedCommand("AddCommand", typeof(FunctionListCommands));

		// Token: 0x0400003D RID: 61
		public static RoutedCommand DeleteCommand = new RoutedCommand("DeleteCommand", typeof(FunctionListCommands));

		// Token: 0x0400003E RID: 62
		public static RoutedCommand CopyCommand = new RoutedCommand("CopyCommand", typeof(FunctionListCommands));

		// Token: 0x0400003F RID: 63
		public static RoutedCommand InsertCommand = new RoutedCommand("CopyCommand", typeof(FunctionListCommands));

		// Token: 0x04000040 RID: 64
		public static RoutedCommand SetFavoriteCommand = new RoutedCommand("SetFavoriteCommand", typeof(FunctionListCommands));
	}
}
