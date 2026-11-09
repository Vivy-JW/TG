using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000036 RID: 54
	public partial class ItemsPropertyEditor : Window
	{
		// Token: 0x0600015C RID: 348 RVA: 0x0000A63C File Offset: 0x0000883C
		public ItemsPropertyEditor()
		{
			this.InitializeComponent();
			base.DataContextChanged += this.ItemsPropertyEditor_DataContextChanged;
			base.Closed += this.ItemsPropertyEditor_Closed;
			base.Closing += this.ItemsPropertyEditor_Closing;
		}

		// Token: 0x17000024 RID: 36
		// (get) Token: 0x0600015D RID: 349 RVA: 0x0000A696 File Offset: 0x00008896
		// (set) Token: 0x0600015E RID: 350 RVA: 0x0000A69E File Offset: 0x0000889E
		public PackageKey ProgramKey { get; set; }

		// Token: 0x17000025 RID: 37
		// (get) Token: 0x0600015F RID: 351 RVA: 0x0000A6A7 File Offset: 0x000088A7
		// (set) Token: 0x06000160 RID: 352 RVA: 0x0000A6AF File Offset: 0x000088AF
		public string ComponentName { get; set; }

		// Token: 0x17000026 RID: 38
		// (get) Token: 0x06000161 RID: 353 RVA: 0x0000A6B8 File Offset: 0x000088B8
		// (set) Token: 0x06000162 RID: 354 RVA: 0x0000A6C0 File Offset: 0x000088C0
		public string Widget { get; set; }

		// Token: 0x06000163 RID: 355 RVA: 0x0000A6D4 File Offset: 0x000088D4
		private void ItemsPropertyEditor_Closing(object sender, CancelEventArgs e)
		{
			ItemData itemData = this.itemSource.Where<ItemData>((ItemData item) => item.IsChanged).FirstOrDefault<ItemData>();
			if (itemData != null)
			{
				if (DesignerMessageBox.Show(Application.Current.FindResource("Message_ConfirmBeforeClose") as string, Application.Current.FindResource("WinTitle_Close") as string, MessageBoxButton.YesNo) == MessageBoxResult.No)
				{
					e.Cancel = true;
					return;
				}
				this.itemSource.Clear();
			}
		}

		// Token: 0x06000164 RID: 356 RVA: 0x0000A758 File Offset: 0x00008958
		private void ItemsPropertyEditor_Closed(object sender, EventArgs e)
		{
			base.DataContextChanged -= this.ItemsPropertyEditor_DataContextChanged;
			base.Closed -= this.ItemsPropertyEditor_Closed;
			base.Closing -= this.ItemsPropertyEditor_Closing;
			this.datagrid1.SetValue(ItemsControl.ItemsSourceProperty, null);
			this.datagrid1.DataContext = null;
		}

		// Token: 0x06000165 RID: 357 RVA: 0x0000A7B8 File Offset: 0x000089B8
		private void ItemsPropertyEditor_DataContextChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			this.datagrid1.CancelEdit();
			if (e.NewValue is XmlElement)
			{
				XmlElement xmlElement = e.NewValue as XmlElement;
				this.UpdateItemsSource(xmlElement.GetItems());
				this.datagrid1.ItemsSource = this.itemSource;
			}
		}

		// Token: 0x06000166 RID: 358 RVA: 0x0000A80C File Offset: 0x00008A0C
		private void UpdateItemsSource(List<XmlElement> items)
		{
			this.oldSource = items;
			this.itemSource.Clear();
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
			foreach (XmlElement xmlElement in items)
			{
				this.itemSource.Add(new ItemData(this.Widget, this.ComponentName, items.Count + 1)
				{
					Program = this.ProgramKey,
					Name = xmlElement.Name,
					Text = xmlElement.Text,
					Description = specificationInfo.GetFieldLocalStringText(xmlElement.Text)
				});
			}
		}

		// Token: 0x06000167 RID: 359 RVA: 0x0000A8D8 File Offset: 0x00008AD8
		private void btnAdd_Click(object sender, RoutedEventArgs e)
		{
			this.datagrid1.CommitEdit();
			this.itemSource.Add(new ItemData(this.Widget, this.ComponentName, this.datagrid1.Items.Count + 1)
			{
				Program = this.ProgramKey
			});
			this.datagrid1.SelectedIndex = this.itemSource.Count<ItemData>() - 1;
			DataGridCell cell = DataGridHelper.GetCell(this.datagrid1, this.itemSource.Count<ItemData>() - 1, 0);
			cell.Focus();
			this.datagrid1.BeginEdit();
		}

		// Token: 0x06000168 RID: 360 RVA: 0x0000A974 File Offset: 0x00008B74
		private void btnDel_Click(object sender, RoutedEventArgs e)
		{
			ItemData itemData = this.datagrid1.SelectedItem as ItemData;
			this.itemSource.Remove(itemData);
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
			specificationInfo.DeleteFieldLocalStringText(itemData.Text);
		}

		// Token: 0x06000169 RID: 361 RVA: 0x0000A9C4 File Offset: 0x00008BC4
		private void btnModify_Click(object sender, RoutedEventArgs e)
		{
			DataGridCell cell = DataGridHelper.GetCell(this.datagrid1, this.datagrid1.SelectedIndex, 0);
			cell.Focus();
			this.datagrid1.BeginEdit();
		}

		// Token: 0x0600016A RID: 362 RVA: 0x0000A9FC File Offset: 0x00008BFC
		private void btnUp_Click(object sender, RoutedEventArgs e)
		{
			this.datagrid1.CommitEdit();
			int selectedIndex = this.datagrid1.SelectedIndex;
			ItemData itemData = this.itemSource[selectedIndex];
			this.itemSource.RemoveAt(selectedIndex);
			this.itemSource.Insert(selectedIndex - 1, itemData);
			this.datagrid1.SelectedIndex = selectedIndex - 1;
		}

		// Token: 0x0600016B RID: 363 RVA: 0x0000AA58 File Offset: 0x00008C58
		private void btnDown_Click(object sender, RoutedEventArgs e)
		{
			this.datagrid1.CommitEdit();
			int selectedIndex = this.datagrid1.SelectedIndex;
			ItemData itemData = this.itemSource[selectedIndex];
			this.itemSource.RemoveAt(selectedIndex);
			this.itemSource.Insert(selectedIndex + 1, itemData);
			this.datagrid1.SelectedIndex = selectedIndex + 1;
		}

		// Token: 0x0600016C RID: 364 RVA: 0x0000AAB3 File Offset: 0x00008CB3
		private void btnOk_Click(object sender, RoutedEventArgs e)
		{
			this.saveChanged();
		}

		// Token: 0x0600016D RID: 365 RVA: 0x0000AAC4 File Offset: 0x00008CC4
		private void btnCancel_Click(object sender, RoutedEventArgs e)
		{
			ItemData itemData = this.itemSource.Where<ItemData>((ItemData item) => item.IsChanged).FirstOrDefault<ItemData>();
			if (itemData != null && DesignerMessageBox.Show(Application.Current.FindResource("Message_ConfirmBeforeClose") as string, Application.Current.FindResource("WinTitle_Close") as string, MessageBoxButton.YesNo) == MessageBoxResult.Yes)
			{
				this.itemSource.Clear();
			}
			base.Close();
		}

		// Token: 0x0600016E RID: 366 RVA: 0x0000AB64 File Offset: 0x00008D64
		private void saveChanged()
		{
			this.datagrid1.CommitEdit();
			List<ItemData> list = this.itemSource.ToList<ItemData>();
			foreach (ItemData itemData in list)
			{
				if (itemData.Name.Trim() == "" && itemData.Text.Trim() == "")
				{
					this.itemSource.Remove(itemData);
				}
				else if (itemData.Name.Trim() == "" && itemData.Text.Trim() != "")
				{
					itemData.Name = itemData.Text;
				}
			}
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
			XmlElement oldItem;
			foreach (XmlElement xmlElement in this.oldSource)
			{
				oldItem = xmlElement;
				if (this.itemSource.Where<ItemData>((ItemData newItem) => oldItem.Text.Equals(newItem.Text)).FirstOrDefault<ItemData>() == null)
				{
					specificationInfo.DeleteFieldLocalStringText(oldItem.Text);
				}
			}
			XmlElement xmlElement2 = base.DataContext as XmlElement;
			List<XmlElement> list2 = new List<XmlElement>();
			foreach (ItemData itemData2 in this.itemSource)
			{
				specificationInfo.SetFieldLocalStringText(itemData2.Text, itemData2.Description);
				XmlElement xmlElement3 = ComponentFactory.CreateEmptyComponent(xmlElement2.Key, ComponentType.Item, itemData2.Name);
				xmlElement3.Text = itemData2.Text;
				xmlElement3.SetAttribute("lstrtext", "true");
				list2.Add(xmlElement3);
				itemData2.IsChanged = false;
			}
			xmlElement2.ReplaceItems(list2);
			base.Close();
		}

		// Token: 0x0600016F RID: 367 RVA: 0x0000AD94 File Offset: 0x00008F94
		private void datagrid1_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			this.datagrid1.CommitEdit();
			this.btnUp.IsEnabled = true;
			this.btnDown.IsEnabled = true;
			if (this.datagrid1.SelectedIndex == 0)
			{
				this.btnUp.IsEnabled = false;
			}
			if (this.datagrid1.SelectedIndex == this.itemSource.Count<ItemData>() - 1)
			{
				this.btnDown.IsEnabled = false;
			}
		}

		// Token: 0x040000A2 RID: 162
		private ObservableCollection<ItemData> itemSource = new ObservableCollection<ItemData>();

		// Token: 0x040000A3 RID: 163
		private List<XmlElement> oldSource;
	}
}
