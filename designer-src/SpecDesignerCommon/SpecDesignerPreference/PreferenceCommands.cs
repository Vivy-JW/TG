using System;
using System.Windows.Controls;
using System.Windows.Input;

namespace SpecDesignerPreference
{
	// Token: 0x02000144 RID: 324
	public class PreferenceCommands
	{
		// Token: 0x170002F2 RID: 754
		// (get) Token: 0x06000B52 RID: 2898 RVA: 0x00037BA3 File Offset: 0x00035DA3
		public static RoutedCommand ShowProgramNoWindowCommand
		{
			get
			{
				if (PreferenceCommands._showProgramNoWindowCommand == null)
				{
					PreferenceCommands._showProgramNoWindowCommand = new RoutedCommand("ShowProgramNoWindowCommand", typeof(PreferenceCommands));
				}
				return PreferenceCommands._showProgramNoWindowCommand;
			}
		}

		// Token: 0x06000B53 RID: 2899 RVA: 0x00037BCA File Offset: 0x00035DCA
		public static void CanProgramNoWindow(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000B54 RID: 2900 RVA: 0x00037BD4 File Offset: 0x00035DD4
		public static void ExecutedProgramNoWindow(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			TextBox textBox = e.Parameter as TextBox;
			if (textBox != null)
			{
				PreferenceCommonUsedWindow.ShowProgramNoWindow(textBox);
			}
		}

		// Token: 0x04000461 RID: 1121
		private static RoutedCommand _showProgramNoWindowCommand;
	}
}
