using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesigner.Controls.Controls;
using SpecDesigner.FormEditor.DBStructure;
using SpecDesigner.FormEditor.Helpers;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using SpecDesignerPreference;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000020 RID: 32
	public partial class WidgetBox : UserControl
	{
		// Token: 0x0600010B RID: 267 RVA: 0x0000647C File Offset: 0x0000467C
		public WidgetBox()
		{
			this.InitializeComponent();
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingFormEvent>().Subscribe(new Action<PackageKey>(this.LoadedWidgetBox));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.TzpFileClosedSubscribe));
			base.CommandBindings.Add(new CommandBinding(WidgetToolbarCommands.AddWidget, new ExecutedRoutedEventHandler(this.ExecutedAdd), new CanExecuteRoutedEventHandler(this.CanExecuteAdd)));
			base.CommandBindings.Add(new CommandBinding(WidgetToolbarCommands.AddWidgetByDataControl, new ExecutedRoutedEventHandler(this.ExecutedAddWidgetByDataControl), new CanExecuteRoutedEventHandler(this.CanExecuteAdd)));
			base.CommandBindings.Add(new CommandBinding(WidgetToolbarCommands.SetTabIndex, new ExecutedRoutedEventHandler(this.ExecutedSetTabIndex)));
			base.CommandBindings.Add(new CommandBinding(WidgetToolbarCommands.AdjustContainerBlankArea, new ExecutedRoutedEventHandler(this.ExecutedAdjustContainerBlankArea)));
		}

		// Token: 0x0600010C RID: 268 RVA: 0x0000656D File Offset: 0x0000476D
		private void CanExecuteSetTabIndex(object sedner, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = SettingManager.Get().Info_ModFd != null;
		}

		// Token: 0x0600010D RID: 269 RVA: 0x0000658C File Offset: 0x0000478C
		private void ExecutedSetTabIndex(object sender, ExecutedRoutedEventArgs e)
		{
			if (!this._isTabIndexShow)
			{
				if (!PreferenceManager.Current.Settings.TabIndexSort)
				{
					MessageBoxResult messageBoxResult = DesignerMessageBox.Show(Application.Current.FindResource("Message_ReorderTabIndex") as string, "TabIndex", MessageBoxButton.YesNo);
					ComponentTabIndexService.Get(this._programKey).Show(messageBoxResult == MessageBoxResult.Yes);
				}
				else
				{
					ComponentTabIndexService.Get(this._programKey).Show(true);
				}
			}
			else
			{
				ComponentTabIndexService.Get(this._programKey).Hide();
			}
			this._isTabIndexShow = !this._isTabIndexShow;
		}

		// Token: 0x0600010E RID: 270 RVA: 0x0000661C File Offset: 0x0000481C
		private void CanExecuteAdd(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = SettingManager.Get().Info_ModFd != null;
		}

		// Token: 0x0600010F RID: 271 RVA: 0x00006634 File Offset: 0x00004834
		private void ExecutedAdd(object sender, ExecutedRoutedEventArgs e)
		{
			string text = e.Parameter as string;
			if (e.Parameter.ToString() == "DateTimeEdit" && SettingManager.Get().ErpVer == "1.0")
			{
				return;
			}
			ComponentType componentType;
			if (Enum.TryParse<ComponentType>(text, out componentType))
			{
				AddWidgetAdornerHelper.Show(this.formWindow.LayoutRoot, componentType, this._programKey);
				return;
			}
			string text2;
			if ((text2 = text) != null)
			{
				if (text2 == "ReferenceLabel")
				{
					AddWidgetAdornerHelper.Show(this.formWindow.LayoutRoot, SpecNodeType.REFERENCE, this._programKey);
					return;
				}
				if (text2 == "MultiLangButtonEdit")
				{
					AddWidgetAdornerHelper.Show(this.formWindow.LayoutRoot, SpecNodeType.MULTILANG, this._programKey);
					return;
				}
				if (!(text2 == "ButtonQuery"))
				{
					return;
				}
				AddWidgetAdornerHelper.Show(this.formWindow.LayoutRoot, SpecNodeType.PROGREL, this._programKey);
			}
		}

		// Token: 0x06000110 RID: 272 RVA: 0x00006714 File Offset: 0x00004914
		private void ExecutedAddWidgetByDataControl(object sender, ExecutedRoutedEventArgs e)
		{
			new DBStructureCreator(this._programKey)
			{
				LayoutEditor = this.formWindow.LayoutRoot
			}.ShowDialog();
		}

		// Token: 0x06000111 RID: 273 RVA: 0x00006745 File Offset: 0x00004945
		private void CanExecuteAdjustContainerBlankArea(object sedner, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = SettingManager.Get().Info_ModFd != null;
		}

		// Token: 0x06000112 RID: 274 RVA: 0x00006764 File Offset: 0x00004964
		private void ExecutedAdjustContainerBlankArea(object sender, ExecutedRoutedEventArgs e)
		{
			if (MessageBox.Show("確認要縮小額外的空白區域？", "確認", MessageBoxButton.OKCancel, MessageBoxImage.Question) == MessageBoxResult.OK)
			{
				this.ShrinkContainer(((SpecificationInfo)this.formWindow.DataContext).FormNode);
				((SpecificationInfo)this.formWindow.DataContext).FormNode["gridHeight"] = "1";
				((SpecificationInfo)this.formWindow.DataContext).FormNode["gridWidth"] = "1";
			}
		}

		// Token: 0x06000113 RID: 275 RVA: 0x000067EC File Offset: 0x000049EC
		private void ShrinkContainer(XmlElement formNode)
		{
			foreach (XmlElement xmlElement in formNode.Nodes)
			{
				if (!(xmlElement.Name == "worksheet") && (xmlElement.Type == ComponentType.Grid || xmlElement.Type == ComponentType.HBox || xmlElement.Type == ComponentType.VBox || xmlElement.Type == ComponentType.Group || xmlElement.Type == ComponentType.Folder || xmlElement.Type == ComponentType.Page))
				{
					this.ShrinkContainer(xmlElement);
					xmlElement["gridHeight"] = "1";
					xmlElement["gridWidth"] = "1";
				}
			}
		}

		// Token: 0x06000114 RID: 276 RVA: 0x000068A4 File Offset: 0x00004AA4
		public void TzpFileClosedSubscribe(PackageKey programKey)
		{
			if (programKey.Equals(this._programKey))
			{
				EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.TzpFileClosedSubscribe));
				foreach (WidgetToolItem widgetToolItem in this.itemsContainer.ToolBars[1].Items.OfType<WidgetToolItem>())
				{
					widgetToolItem.Dispose();
				}
				foreach (WidgetToolItem widgetToolItem2 in this.itemsContainer.ToolBars[0].Items.OfType<WidgetToolItem>())
				{
					widgetToolItem2.Dispose();
				}
				this.itemsContainer.ToolBars.Clear();
			}
		}

		// Token: 0x06000115 RID: 277 RVA: 0x00006994 File Offset: 0x00004B94
		private void LoadedWidgetBox(PackageKey progKey)
		{
			this._programKey = progKey;
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingFormEvent>().Unsubscribe(new Action<PackageKey>(this.LoadedWidgetBox));
			this.formWindow = VisualTreeHelperEx.FindLogicVisualParent1<FormEditorMainWindow>(this);
			if (SettingManager.Get().Info_ModFd == null)
			{
				this.itemsContainer.IsEnabled = false;
			}
		}

		// Token: 0x04000095 RID: 149
		private FormEditorMainWindow formWindow;

		// Token: 0x04000096 RID: 150
		private PackageKey _programKey;

		// Token: 0x04000097 RID: 151
		private bool _isTabIndexShow;
	}
}
