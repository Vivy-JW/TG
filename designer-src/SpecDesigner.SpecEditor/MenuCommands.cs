using System;
using System.Windows.Input;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor
{
	// Token: 0x02000041 RID: 65
	public class MenuCommands
	{
		// Token: 0x17000030 RID: 48
		// (get) Token: 0x060001B1 RID: 433 RVA: 0x0000C174 File Offset: 0x0000A374
		public static RoutedCommand SetCiteCommand
		{
			get
			{
				if (MenuCommands._setCiteCommand == null)
				{
					MenuCommands._setCiteCommand = new RoutedCommand("SetCiteCommand", typeof(MenuCommands));
				}
				return MenuCommands._setCiteCommand;
			}
		}

		// Token: 0x060001B2 RID: 434 RVA: 0x0000C19C File Offset: 0x0000A39C
		public static void CanExecuteSetCite(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			bool flag = false;
			SpecProgram specProgram = e.Parameter as SpecProgram;
			if (specProgram != null && SettingManager.Get().GetTzpManger(specProgram.ProgramKey) != null)
			{
				XElement citedSpec = specProgram.CitedSpec;
				flag = citedSpec != null;
			}
			e.CanExecute = flag;
		}

		// Token: 0x040000CD RID: 205
		private static RoutedCommand _setCiteCommand;
	}
}
