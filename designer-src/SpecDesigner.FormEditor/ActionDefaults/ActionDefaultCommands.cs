using System;
using System.Windows;
using System.Windows.Input;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.ActionDefaults
{
	// Token: 0x02000027 RID: 39
	public class ActionDefaultCommands
	{
		// Token: 0x17000047 RID: 71
		// (get) Token: 0x0600015A RID: 346 RVA: 0x000077FA File Offset: 0x000059FA
		public static RoutedCommand AddActionCommand
		{
			get
			{
				if (ActionDefaultCommands._addActionCommand == null)
				{
					ActionDefaultCommands._addActionCommand = new RoutedCommand();
				}
				return ActionDefaultCommands._addActionCommand;
			}
		}

		// Token: 0x17000048 RID: 72
		// (get) Token: 0x0600015B RID: 347 RVA: 0x00007812 File Offset: 0x00005A12
		public static RoutedCommand DeleteActionCommand
		{
			get
			{
				if (ActionDefaultCommands._deleteActionCommand == null)
				{
					ActionDefaultCommands._deleteActionCommand = new RoutedCommand();
				}
				return ActionDefaultCommands._deleteActionCommand;
			}
		}

		// Token: 0x0600015C RID: 348 RVA: 0x0000782C File Offset: 0x00005A2C
		public static void CanExecuteDeleteAction(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			bool flag = false;
			SpecActionNode specActionNode = e.Parameter as SpecActionNode;
			if (specActionNode != null)
			{
				TzpManager tzpManger = SettingManager.Get().GetTzpManger(specActionNode.ProgramKey);
				if (tzpManger != null)
				{
					SpecificationInfo specificationInfo = tzpManger.SpecificationInfo;
					flag = !specActionNode.IsActionDefaults && specificationInfo.FindNodeByName(specActionNode.Name) == null;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x0600015D RID: 349 RVA: 0x00007894 File Offset: 0x00005A94
		public static void ExecutedDeleteAction(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SpecActionNode specActionNode = e.Parameter as SpecActionNode;
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(specActionNode.ProgramKey).SpecificationInfo;
			string text = (string)Application.Current.FindResource("ads_ActionDelConfirm");
			string text2 = (string)Application.Current.FindResource("WinTitle_Delete");
			if (DesignerMessageBox.Show(string.Format(text, specActionNode.LocalString), text2, MessageBoxButton.YesNo) == MessageBoxResult.Yes)
			{
				DeleteActUndoRedoCommand deleteActUndoRedoCommand = new DeleteActUndoRedoCommand(specActionNode);
				SettingManager.Get().GetUndoRedoManager(specActionNode.ProgramKey).AddThenExecute(deleteActUndoRedoCommand);
			}
		}

		// Token: 0x040000BE RID: 190
		private static RoutedCommand _addActionCommand;

		// Token: 0x040000BF RID: 191
		private static RoutedCommand _deleteActionCommand;
	}
}
