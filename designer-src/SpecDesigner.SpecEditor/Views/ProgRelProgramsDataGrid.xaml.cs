using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x0200004D RID: 77
	public partial class ProgRelProgramsDataGrid : UserControl
	{
		// Token: 0x060001D8 RID: 472 RVA: 0x0000C885 File Offset: 0x0000AA85
		public ProgRelProgramsDataGrid()
		{
			this.InitializeComponent();
			base.CommandBindings.Add(new CommandBinding(SpecPropertyCommands.ShowZoomsWindowCommand, new ExecutedRoutedEventHandler(SpecPropertyCommands.ExecutedShowZoomsWindow), new CanExecuteRoutedEventHandler(SpecPropertyCommands.CanShowZoomsWindow)));
		}

		// Token: 0x17000033 RID: 51
		// (get) Token: 0x060001D9 RID: 473 RVA: 0x0000C8C1 File Offset: 0x0000AAC1
		// (set) Token: 0x060001DA RID: 474 RVA: 0x0000C8D3 File Offset: 0x0000AAD3
		public SpecProgRelNode ProgRelNode
		{
			get
			{
				return (SpecProgRelNode)base.GetValue(ProgRelProgramsDataGrid.ProgRelNodeProperty);
			}
			set
			{
				base.SetValue(ProgRelProgramsDataGrid.ProgRelNodeProperty, value);
			}
		}

		// Token: 0x060001DB RID: 475 RVA: 0x0000C8E1 File Offset: 0x0000AAE1
		public static void SpecProgRelNodeChanged(DependencyObject obj, DependencyPropertyChangedEventArgs args)
		{
		}

		// Token: 0x060001DC RID: 476 RVA: 0x0000C8E4 File Offset: 0x0000AAE4
		private void Add_Click(object sender, RoutedEventArgs e)
		{
			ProgRelProgram progRelProgram = ProgRelProgram.Create(this.ProgRelNode);
			ChangeProgRelProgramUndoRedoCommand changeProgRelProgramUndoRedoCommand = new ChangeProgRelProgramUndoRedoCommand(this.ProgRelNode, progRelProgram, false);
			SettingManager.Get().GetUndoRedoManager(this.ProgRelNode.ProgramKey).AddThenExecute(changeProgRelProgramUndoRedoCommand);
		}

		// Token: 0x060001DD RID: 477 RVA: 0x0000C928 File Offset: 0x0000AB28
		private void Delete_Click(object sender, RoutedEventArgs e)
		{
			if (this.ProgramsDataGrid.SelectedItem != null)
			{
				ProgRelProgram progRelProgram = this.ProgramsDataGrid.SelectedItem as ProgRelProgram;
				if (progRelProgram != null)
				{
					string text = Application.Current.FindResource("ads_ActionDelConfirm") as string;
					text = string.Format(text, progRelProgram.Program);
					if (DesignerMessageBox.Show(text, "Delete", MessageBoxButton.YesNo) == MessageBoxResult.Yes && progRelProgram != null)
					{
						ChangeProgRelProgramUndoRedoCommand changeProgRelProgramUndoRedoCommand = new ChangeProgRelProgramUndoRedoCommand(this.ProgRelNode, progRelProgram, true);
						SettingManager.Get().GetUndoRedoManager(this.ProgRelNode.ProgramKey).AddThenExecute(changeProgRelProgramUndoRedoCommand);
					}
				}
			}
			this.ProgramsDataGrid.CommitEdit();
		}

		// Token: 0x040000D0 RID: 208
		public static readonly DependencyProperty ProgRelNodeProperty = DependencyProperty.Register("ProgRelNode", typeof(SpecProgRelNode), typeof(ProgRelProgramsDataGrid), new FrameworkPropertyMetadata(new PropertyChangedCallback(ProgRelProgramsDataGrid.SpecProgRelNodeChanged)));
	}
}
