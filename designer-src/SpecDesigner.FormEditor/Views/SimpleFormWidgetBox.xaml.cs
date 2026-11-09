using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesignerCommon;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000059 RID: 89
	public partial class SimpleFormWidgetBox : UserControl
	{
		// Token: 0x0600037F RID: 895 RVA: 0x000132CC File Offset: 0x000114CC
		public SimpleFormWidgetBox()
		{
			this.InitializeComponent();
			base.CommandBindings.Add(new CommandBinding(SimpleFormWidgetToolbarCommands.QuickHide, new ExecutedRoutedEventHandler(this.ExecuteQuickHide)));
			base.CommandBindings.Add(new CommandBinding(SimpleFormWidgetToolbarCommands.QuickShow, new ExecutedRoutedEventHandler(this.ExecuteQuickShow)));
			base.CommandBindings.Add(new CommandBinding(SimpleFormWidgetToolbarCommands.HideAll, new ExecutedRoutedEventHandler(this.ExecuteHideAll)));
			base.CommandBindings.Add(new CommandBinding(SimpleFormWidgetToolbarCommands.ShowAll, new ExecutedRoutedEventHandler(this.ExecuteShowAll)));
		}

		// Token: 0x06000380 RID: 896 RVA: 0x00013370 File Offset: 0x00011570
		private void ExecuteQuickHide(object sender, ExecutedRoutedEventArgs e)
		{
			if (this.ChkBoxQHide.IsChecked == true)
			{
				this.ChkBoxQShow.IsChecked = new bool?(false);
			}
			if (this.ChkBoxQHide.IsChecked == true && ManagedForm.Current != null)
			{
				ManagedForm.Current.CurrentSimpleFormSetupType = SimpleFormSetupType.Hide;
			}
			this.CheckSimpleFormStopSetup();
		}

		// Token: 0x06000381 RID: 897 RVA: 0x000133E8 File Offset: 0x000115E8
		private void ExecuteQuickShow(object sender, ExecutedRoutedEventArgs e)
		{
			if (this.ChkBoxQHide.IsChecked == true)
			{
				this.ChkBoxQHide.IsChecked = new bool?(false);
			}
			if (this.ChkBoxQShow.IsChecked == true && ManagedForm.Current != null)
			{
				ManagedForm.Current.CurrentSimpleFormSetupType = SimpleFormSetupType.Show;
			}
			this.CheckSimpleFormStopSetup();
		}

		// Token: 0x06000382 RID: 898 RVA: 0x00013460 File Offset: 0x00011660
		private void ExecuteHideAll(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			if (ManagedForm.Current != null)
			{
				ManagedForm.Current.SimpleFormHideAll(true);
			}
		}

		// Token: 0x06000383 RID: 899 RVA: 0x0001347B File Offset: 0x0001167B
		private void ExecuteShowAll(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			if (ManagedForm.Current != null)
			{
				ManagedForm.Current.SimpleFormHideAll(false);
			}
		}

		// Token: 0x06000384 RID: 900 RVA: 0x00013498 File Offset: 0x00011698
		private void CheckSimpleFormStopSetup()
		{
			if (this.ChkBoxQShow.IsChecked == false && this.ChkBoxQHide.IsChecked == false)
			{
				ManagedForm.Current.CurrentSimpleFormSetupType = SimpleFormSetupType.None;
			}
		}
	}
}
