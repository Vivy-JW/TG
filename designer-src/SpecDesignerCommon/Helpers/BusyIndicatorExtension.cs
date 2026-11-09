using System;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x020000ED RID: 237
	public static class BusyIndicatorExtension
	{
		// Token: 0x060007EE RID: 2030 RVA: 0x000236D0 File Offset: 0x000218D0
		public static void IsBusy(bool isBusy)
		{
			Grid grid = Application.Current.MainWindow.FindName("busyIndicator") as Grid;
			if (grid != null)
			{
				grid.Visibility = (isBusy ? Visibility.Visible : Visibility.Collapsed);
			}
		}
	}
}
