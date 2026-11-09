using System;
using System.Windows.Input;

namespace SpecDesigner.Search
{
	// Token: 0x02000009 RID: 9
	public static class SearchCommands
	{
		// Token: 0x1700001B RID: 27
		// (get) Token: 0x06000089 RID: 137 RVA: 0x00003F2A File Offset: 0x0000212A
		public static RoutedCommand SearchCommnad
		{
			get
			{
				if (SearchCommands._searchCommnad == null)
				{
					SearchCommands._searchCommnad = new RoutedCommand();
				}
				return SearchCommands._searchCommnad;
			}
		}

		// Token: 0x1700001C RID: 28
		// (get) Token: 0x0600008A RID: 138 RVA: 0x00003F42 File Offset: 0x00002142
		public static RoutedCommand PreviousCommnad
		{
			get
			{
				if (SearchCommands._previousCommand == null)
				{
					SearchCommands._previousCommand = new RoutedCommand();
				}
				return SearchCommands._previousCommand;
			}
		}

		// Token: 0x1700001D RID: 29
		// (get) Token: 0x0600008B RID: 139 RVA: 0x00003F5A File Offset: 0x0000215A
		public static RoutedCommand NextCommnad
		{
			get
			{
				if (SearchCommands._nextCommnad == null)
				{
					SearchCommands._nextCommnad = new RoutedCommand();
				}
				return SearchCommands._nextCommnad;
			}
		}

		// Token: 0x04000030 RID: 48
		private static RoutedCommand _searchCommnad;

		// Token: 0x04000031 RID: 49
		private static RoutedCommand _previousCommand;

		// Token: 0x04000032 RID: 50
		private static RoutedCommand _nextCommnad;
	}
}
