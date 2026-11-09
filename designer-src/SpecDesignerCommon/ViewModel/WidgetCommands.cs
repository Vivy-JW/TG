using System;
using System.Windows.Input;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200007D RID: 125
	public static class WidgetCommands
	{
		// Token: 0x17000159 RID: 345
		// (get) Token: 0x060004DC RID: 1244 RVA: 0x00015E22 File Offset: 0x00014022
		public static RoutedCommand DeleteWidgetCommand
		{
			get
			{
				if (WidgetCommands._deleteWidgetCommand == null)
				{
					WidgetCommands._deleteWidgetCommand = new RoutedCommand("DeleteWidgetCommand", typeof(WidgetCommands));
				}
				return WidgetCommands._deleteWidgetCommand;
			}
		}

		// Token: 0x1700015A RID: 346
		// (get) Token: 0x060004DD RID: 1245 RVA: 0x00015E49 File Offset: 0x00014049
		public static RoutedCommand ConvertWidgetCommand
		{
			get
			{
				if (WidgetCommands._convertWidgetCommand == null)
				{
					WidgetCommands._convertWidgetCommand = new RoutedCommand("ConvertWidgetCommand", typeof(WidgetCommands));
				}
				return WidgetCommands._convertWidgetCommand;
			}
		}

		// Token: 0x1700015B RID: 347
		// (get) Token: 0x060004DE RID: 1246 RVA: 0x00015E70 File Offset: 0x00014070
		public static RoutedCommand MoveWidgetCommand
		{
			get
			{
				if (WidgetCommands._moveWidgetCommand == null)
				{
					WidgetCommands._moveWidgetCommand = new RoutedCommand("MoveWidgetCommand", typeof(WidgetCommands));
				}
				return WidgetCommands._moveWidgetCommand;
			}
		}

		// Token: 0x1700015C RID: 348
		// (get) Token: 0x060004DF RID: 1247 RVA: 0x00015E97 File Offset: 0x00014097
		public static RoutedCommand AddChildCommand
		{
			get
			{
				if (WidgetCommands._addChildCommand == null)
				{
					WidgetCommands._addChildCommand = new RoutedCommand("AddChildCommand", typeof(WidgetCommands));
				}
				return WidgetCommands._addChildCommand;
			}
		}

		// Token: 0x040001CD RID: 461
		private static RoutedCommand _deleteWidgetCommand;

		// Token: 0x040001CE RID: 462
		private static RoutedCommand _convertWidgetCommand;

		// Token: 0x040001CF RID: 463
		private static RoutedCommand _moveWidgetCommand;

		// Token: 0x040001D0 RID: 464
		private static RoutedCommand _addChildCommand;
	}
}
