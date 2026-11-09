using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using Microsoft.Expression.Shapes;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.ActionDefaults
{
	// Token: 0x02000058 RID: 88
	public partial class ActionDefault : UserControl, IDisposable
	{
		// Token: 0x06000378 RID: 888 RVA: 0x0001306C File Offset: 0x0001126C
		public ActionDefault()
		{
			this.InitializeComponent();
			base.Cursor = Cursors.Hand;
			base.MouseEnter += this.ActionDefault_MouseEnter;
			base.MouseLeave += this.ActionDefault_MouseLeave;
			this.actionBtn.MouseLeftButtonDown += this.actionBtn_MouseLeftButtonDown;
		}

		// Token: 0x06000379 RID: 889 RVA: 0x000130CC File Offset: 0x000112CC
		private void actionBtn_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			Keyboard.Focus(sender as UIElement);
			SpecActionNode specActionNode = base.DataContext as SpecActionNode;
			if (specActionNode == null)
			{
				return;
			}
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(specActionNode.ProgramKey).SpecificationInfo.FindNodeByName(specActionNode.Name);
			if (formSpecModel != null && formSpecModel.GeneroComponent != null)
			{
				ComponentHelper.Get(specActionNode.ProgramKey).AddSelection(formSpecModel.GeneroComponent, false);
				return;
			}
			ComponentHelper.Get(specActionNode.ProgramKey).SelectAction(specActionNode.Name, specActionNode.ProgramKey);
		}

		// Token: 0x0600037A RID: 890 RVA: 0x00013155 File Offset: 0x00011355
		private void ActionDefault_MouseEnter(object sender, MouseEventArgs e)
		{
			e.Handled = true;
			this.operationButtons.Visibility = Visibility.Visible;
		}

		// Token: 0x0600037B RID: 891 RVA: 0x0001316A File Offset: 0x0001136A
		private void ActionDefault_MouseLeave(object sender, MouseEventArgs e)
		{
			e.Handled = true;
			this.operationButtons.Visibility = Visibility.Collapsed;
		}

		// Token: 0x0600037C RID: 892 RVA: 0x00013180 File Offset: 0x00011380
		public void Dispose()
		{
			base.DataContext = Binding.DoNothing;
			base.CommandBindings.Clear();
			this.addButton.CommandBindings.Clear();
			this.removeButton.CommandBindings.Clear();
			base.MouseEnter -= this.ActionDefault_MouseEnter;
			base.MouseLeave -= this.ActionDefault_MouseLeave;
			this.actionBtn.MouseLeftButtonDown -= this.actionBtn_MouseLeftButtonDown;
			BindingOperations.ClearAllBindings(this);
		}
	}
}
