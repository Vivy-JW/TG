using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000031 RID: 49
	public partial class SpecFieldTextControl : UserControl, IDisposable
	{
		// Token: 0x14000004 RID: 4
		// (add) Token: 0x0600013A RID: 314 RVA: 0x00009EDC File Offset: 0x000080DC
		// (remove) Token: 0x0600013B RID: 315 RVA: 0x00009F14 File Offset: 0x00008114
		public event SpecFieldTextControl.SpecFieldSelectedEventHandler TextIDSelected;

		// Token: 0x0600013C RID: 316 RVA: 0x00009F58 File Offset: 0x00008158
		public SpecFieldTextControl(PackageKey programKey, string defaultText, SpecNodeType type)
		{
			this.InitializeComponent();
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(programKey).SpecificationInfo;
			if (type == SpecNodeType.ACTION)
			{
				this.dataGrid.ItemsSource = specificationInfo.ActionStrings;
			}
			else
			{
				this.dataGrid.ItemsSource = specificationInfo.FieldStrings;
			}
			this.dataGrid.CellEditEnding += this.dataGrid_CellEditEnding;
			if (!string.IsNullOrEmpty(defaultText))
			{
				foreach (object obj in this.dataGrid.ItemsSource)
				{
					AbstractStringNode abstractStringNode = (AbstractStringNode)obj;
					if (defaultText.Equals(abstractStringNode.Name))
					{
						this.dataGrid.SelectedItem = abstractStringNode;
						this.dataGrid.ScrollIntoView(abstractStringNode);
						break;
					}
				}
			}
			base.Loaded += delegate(object sender, RoutedEventArgs e)
			{
				this.MoveFocus(new TraversalRequest(FocusNavigationDirection.Next));
			};
		}

		// Token: 0x0600013D RID: 317 RVA: 0x0000A058 File Offset: 0x00008258
		private void textbox_CellTextBoxLostFocus(object sender, KeyboardFocusChangedEventArgs e)
		{
			this.dataGrid.CommitEdit();
		}

		// Token: 0x0600013E RID: 318 RVA: 0x0000A068 File Offset: 0x00008268
		private void dataGrid_CellEditEnding(object sender, DataGridCellEditEndingEventArgs e)
		{
			if (e.EditAction == DataGridEditAction.Commit)
			{
				AbstractStringNode abstractStringNode = this.dataGrid.SelectedItem as AbstractStringNode;
				abstractStringNode.Status |= SpecStatus.MODIFY;
			}
		}

		// Token: 0x0600013F RID: 319 RVA: 0x0000A0A0 File Offset: 0x000082A0
		private void dataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			if (this.dataGrid.SelectedItem != null)
			{
				AbstractStringNode abstractStringNode = this.dataGrid.SelectedItem as AbstractStringNode;
				if (this.TextIDSelected != null)
				{
					this.TextIDSelected(this, abstractStringNode);
				}
			}
		}

		// Token: 0x06000140 RID: 320 RVA: 0x0000A0E0 File Offset: 0x000082E0
		private void DeleteButtonClicked(object sender, RoutedEventArgs e)
		{
			(this.dataGrid.SelectedItem as AbstractStringNode).Status |= SpecStatus.DELETE;
		}

		// Token: 0x06000141 RID: 321 RVA: 0x0000A0FF File Offset: 0x000082FF
		private void RestoreButtonClicked(object sender, RoutedEventArgs e)
		{
			(this.dataGrid.SelectedItem as AbstractStringNode).Status &= ~SpecStatus.DELETE;
		}

		// Token: 0x06000142 RID: 322 RVA: 0x0000A120 File Offset: 0x00008320
		private void ApplyButtonClicked(object sender, RoutedEventArgs e)
		{
			if (this.dataGrid.SelectedItem != null)
			{
				AbstractStringNode abstractStringNode = this.dataGrid.SelectedItem as AbstractStringNode;
				if (this.TextIDSelected != null)
				{
					this.TextIDSelected(this, abstractStringNode);
				}
			}
		}

		// Token: 0x06000143 RID: 323 RVA: 0x0000A160 File Offset: 0x00008360
		public void Dispose()
		{
			this.TextIDSelected = null;
			this.dataGrid.DataContext = Binding.DoNothing;
		}

		// Token: 0x06000146 RID: 326 RVA: 0x0000A1D4 File Offset: 0x000083D4
		[EditorBrowsable(EditorBrowsableState.Never)]
		[DebuggerNonUserCode]
		[GeneratedCode("PresentationBuildTasks", "4.0.0.0")]
		void IStyleConnector.Connect(int connectionId, object target)
		{
			switch (connectionId)
			{
			case 2:
				((Button)target).Click += this.ApplyButtonClicked;
				return;
			case 3:
				((Button)target).Click += this.DeleteButtonClicked;
				return;
			case 4:
				((Button)target).Click += this.RestoreButtonClicked;
				return;
			default:
				return;
			}
		}

		// Token: 0x02000032 RID: 50
		// (Invoke) Token: 0x06000149 RID: 329
		public delegate void SpecFieldSelectedEventHandler(object sender, AbstractStringNode node);
	}
}
