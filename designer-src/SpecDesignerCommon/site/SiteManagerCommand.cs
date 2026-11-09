using System;
using System.Windows.Input;

namespace SpecDesignerCommon.Site
{
	// Token: 0x02000102 RID: 258
	public class SiteManagerCommand
	{
		// Token: 0x17000272 RID: 626
		// (get) Token: 0x06000914 RID: 2324 RVA: 0x0002BF54 File Offset: 0x0002A154
		public static RoutedCommand ExportCommand
		{
			get
			{
				if (SiteManagerCommand._exportCommand == null)
				{
					SiteManagerCommand._exportCommand = new RoutedCommand("ExportCommand", typeof(SiteManagerCommand));
				}
				return SiteManagerCommand._exportCommand;
			}
		}

		// Token: 0x17000273 RID: 627
		// (get) Token: 0x06000915 RID: 2325 RVA: 0x0002BF7B File Offset: 0x0002A17B
		public static RoutedCommand ImportCommand
		{
			get
			{
				if (SiteManagerCommand._importCommand == null)
				{
					SiteManagerCommand._importCommand = new RoutedCommand("ImportCommand", typeof(SiteManagerCommand));
				}
				return SiteManagerCommand._importCommand;
			}
		}

		// Token: 0x04000339 RID: 825
		private static RoutedCommand _exportCommand;

		// Token: 0x0400033A RID: 826
		private static RoutedCommand _importCommand;
	}
}
