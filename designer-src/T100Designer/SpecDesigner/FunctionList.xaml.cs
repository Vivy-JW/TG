using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Model;
using SpecDesigner.ViewModels;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner
{
	// Token: 0x0200000E RID: 14
	public partial class FunctionList : UserControl
	{
		// Token: 0x06000070 RID: 112 RVA: 0x00003402 File Offset: 0x00001602
		public FunctionList()
		{
			this.InitializeComponent();
			this.BindMenuCommands();
			base.DataContext = ResourceController.GetInstance().GeneralFunctionSuggestion;
		}

		// Token: 0x06000071 RID: 113 RVA: 0x00003428 File Offset: 0x00001628
		private void BindMenuCommands()
		{
			base.CommandBindings.Add(new CommandBinding(FunctionListCommands.SearchCommand, new ExecutedRoutedEventHandler(this.ExecutedSearch)));
			base.CommandBindings.Add(new CommandBinding(FunctionListCommands.CopyCommand, new ExecutedRoutedEventHandler(this.ExecutedCopy)));
			base.CommandBindings.Add(new CommandBinding(FunctionListCommands.DeleteCommand, new ExecutedRoutedEventHandler(this.ExecutedDelete), new CanExecuteRoutedEventHandler(this.CanExecuteDelete)));
			base.CommandBindings.Add(new CommandBinding(FunctionListCommands.AddCommand, new ExecutedRoutedEventHandler(this.ExecutedAdd), new CanExecuteRoutedEventHandler(this.CanExecuteAdd)));
			base.CommandBindings.Add(new CommandBinding(FunctionListCommands.InsertCommand, new ExecutedRoutedEventHandler(this.ExecutedInsert), new CanExecuteRoutedEventHandler(this.CanExecuteInsert)));
		}

		// Token: 0x06000072 RID: 114 RVA: 0x00003504 File Offset: 0x00001704
		private IIntellisenseModel GetSelectedItem()
		{
			return this.contentList.SelectedItem as IIntellisenseModel;
		}

		// Token: 0x06000073 RID: 115 RVA: 0x00003524 File Offset: 0x00001724
		private void ExecutedCopy(object sender, ExecutedRoutedEventArgs e)
		{
			IIntellisenseModel selectedItem = this.GetSelectedItem();
			if (selectedItem == null)
			{
				return;
			}
			Clipboard.Clear();
			Clipboard.SetDataObject(selectedItem.Name);
		}

		// Token: 0x06000074 RID: 116 RVA: 0x0000354C File Offset: 0x0000174C
		private void ExecutedSearch(object sender, ExecutedRoutedEventArgs e)
		{
			string text = e.Parameter as string;
			this.OnSearch(text);
		}

		// Token: 0x06000075 RID: 117 RVA: 0x0000356C File Offset: 0x0000176C
		private void TextBox_TextChanged(object sender, TextChangedEventArgs e)
		{
			this.OnSearch(this.searchBox.Text);
		}

		// Token: 0x06000076 RID: 118 RVA: 0x000035DC File Offset: 0x000017DC
		private void OnSearch(string target)
		{
			target = target.Trim();
			if (string.IsNullOrEmpty(target))
			{
				if (this._customerView != null)
				{
					this._customerView.Filter = null;
				}
				return;
			}
			this._customerView = CollectionViewSource.GetDefaultView(this.contentList.ItemsSource);
			this._customerView.Filter = delegate(object item)
			{
				IIntellisenseModel intellisenseModel = (IIntellisenseModel)item;
				return intellisenseModel != null && intellisenseModel.FullName != null && (intellisenseModel.FullName.Contains(target) || (intellisenseModel.Description != null && intellisenseModel.Description.Contains(target)));
			};
		}

		// Token: 0x06000077 RID: 119 RVA: 0x00003658 File Offset: 0x00001858
		private void CanExecuteDelete(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			IIntellisenseModel intellisenseModel = e.Parameter as IIntellisenseModel;
			bool flag = intellisenseModel != null && intellisenseModel.Type == IntellisenseEnum.Self;
			e.CanExecute = flag;
		}

		// Token: 0x06000078 RID: 120 RVA: 0x00003694 File Offset: 0x00001894
		private void ExecutedDelete(object sender, ExecutedRoutedEventArgs e)
		{
			IIntellisenseModel selectedItem = this.GetSelectedItem();
			if (selectedItem == null)
			{
				return;
			}
			if (selectedItem.Type != IntellisenseEnum.Self)
			{
				return;
			}
			ResourceController.GetInstance().DeleteSuggestion(selectedItem.Name);
		}

		// Token: 0x06000079 RID: 121 RVA: 0x000036E4 File Offset: 0x000018E4
		private void CanExecuteAdd(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			string text = this.createBox.Text;
			bool flag;
			if (!string.IsNullOrWhiteSpace(text))
			{
				IEnumerable<IIntellisenseModel> enumerable = ResourceController.GetInstance().GeneralFunctionSuggestion.Where<IIntellisenseModel>((IIntellisenseModel m) => m.Name == text);
				flag = enumerable.Count<IIntellisenseModel>() <= 0;
			}
			else
			{
				flag = true;
			}
			e.CanExecute = flag;
		}

		// Token: 0x0600007A RID: 122 RVA: 0x0000375C File Offset: 0x0000195C
		private void ExecutedAdd(object sender, ExecutedRoutedEventArgs e)
		{
			string text = this.createBox.Text;
			SelfIntellisenseModel selfIntellisenseModel = new SelfIntellisenseModel(text);
			if (selfIntellisenseModel == null)
			{
				return;
			}
			ResourceController.GetInstance().AddSuggestion(selfIntellisenseModel);
		}

		// Token: 0x0600007B RID: 123 RVA: 0x0000378B File Offset: 0x0000198B
		private void CanExecuteInsert(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = EditorWorkspace.This.ActiveDocument is CodeViewModel || string.IsNullOrWhiteSpace(e.Parameter as string);
		}

		// Token: 0x0600007C RID: 124 RVA: 0x000037C0 File Offset: 0x000019C0
		private void ExecutedInsert(object sender, ExecutedRoutedEventArgs e)
		{
			InsertionCodeEventArgs e2 = new InsertionCodeEventArgs
			{
				ProgramKey = EditorWorkspace.This.ActiveDocument.Key,
				Content = (e.Parameter as string)
			};
			EventAggregatorManager.Global.GetEvent<InsertionCodeEvent>().Publish(e2);
		}

		// Token: 0x0600007D RID: 125 RVA: 0x0000380B File Offset: 0x00001A0B
		private void CanExecuteSetFavorite(object sender, CanExecuteRoutedEventArgs e)
		{
		}

		// Token: 0x0600007E RID: 126 RVA: 0x0000380D File Offset: 0x00001A0D
		private void ExecutedSetFavorite(object sender, CanExecuteRoutedEventArgs e)
		{
		}

		// Token: 0x04000036 RID: 54
		private ICollectionView _customerView;
	}
}
