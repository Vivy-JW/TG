using System;
using System.Windows.Input;

namespace SpecDesigner.Search
{
	// Token: 0x02000008 RID: 8
	public static class ReplaceCommands
	{
		// Token: 0x17000019 RID: 25
		// (get) Token: 0x06000086 RID: 134 RVA: 0x00003EF8 File Offset: 0x000020F8
		public static RoutedCommand ReplaceCommnad
		{
			get
			{
				if (ReplaceCommands._replaceCommnad == null)
				{
					ReplaceCommands._replaceCommnad = new RoutedCommand();
				}
				return ReplaceCommands._replaceCommnad;
			}
		}

		// Token: 0x1700001A RID: 26
		// (get) Token: 0x06000087 RID: 135 RVA: 0x00003F10 File Offset: 0x00002110
		public static RoutedCommand ReplaceAllCommnad
		{
			get
			{
				if (ReplaceCommands._replaceAllCommnad == null)
				{
					ReplaceCommands._replaceAllCommnad = new RoutedCommand();
				}
				return ReplaceCommands._replaceAllCommnad;
			}
		}

		// Token: 0x0400002E RID: 46
		private static RoutedCommand _replaceCommnad;

		// Token: 0x0400002F RID: 47
		private static RoutedCommand _replaceAllCommnad;
	}
}
