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
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Model;
using SpecDesigner.ViewModels;
using SpecDesignerCommon;

namespace SpecDesigner
{
	// Token: 0x02000010 RID: 16
	public partial class CustomAdpListWindow : Window
	{
		// Token: 0x06000082 RID: 130 RVA: 0x0000394F File Offset: 0x00001B4F
		private CustomAdpListWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000083 RID: 131 RVA: 0x000039E8 File Offset: 0x00001BE8
		public CustomAdpListWindow(CustomAdpListWindow.ContentType contentType)
			: this()
		{
			PackageKey packageKey = Application.Current.MainWindow.Tag as PackageKey;
			if (null == packageKey)
			{
				return;
			}
			SettingManager.Get().SaveSetting(packageKey);
			switch (contentType)
			{
			case CustomAdpListWindow.ContentType.src:
			{
				this.click_title.Visibility = Visibility.Visible;
				base.Title = string.Format("{0} [{1}]", Application.Current.FindResource("menu_ShowProgramCustomAdpList") as string, packageKey.Program);
				IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(Application.Current.MainWindow.Tag as PackageKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.SRC == "c" && ap.Status != Status.CREATE && (ap.Status & Status.DELETE) != Status.DELETE);
				base.DataContext = enumerable;
				return;
			}
			case CustomAdpListWindow.ContentType.cite:
			{
				this.click_title.Visibility = Visibility.Visible;
				base.Title = string.Format("{0} [{1}]", Application.Current.FindResource("menu_ShowProgramUncited") as string, packageKey.Program);
				IEnumerable<AddPointModel> enumerable2 = ResourceController.GetInstance().GetProgramInfo(Application.Current.MainWindow.Tag as PackageKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.CiteSetting == YesNo.N && ap.Status != Status.CREATE && (ap.Status & Status.DELETE) != Status.DELETE);
				base.DataContext = enumerable2;
				return;
			}
			case CustomAdpListWindow.ContentType.section:
			{
				this.click_title.Visibility = Visibility.Collapsed;
				base.Title = string.Format("{0} [{1}]", Application.Current.FindResource("menu_ShowProgramSectionUncited") as string, packageKey.Program);
				IEnumerable<SectionModel> enumerable3 = ResourceController.GetInstance().GetProgramInfo(Application.Current.MainWindow.Tag as PackageKey).Sections.Where<SectionModel>((SectionModel section) => (section.SRC == "m" || section.SRC == "c") && section.Status != "d");
				base.DataContext = enumerable3;
				return;
			}
			default:
				return;
			}
		}

		// Token: 0x06000084 RID: 132 RVA: 0x00003BC8 File Offset: 0x00001DC8
		private void dataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			AddPointModel addPointModel = this.dataGrid.SelectedItem as AddPointModel;
			if (addPointModel == null)
			{
				return;
			}
			CodeViewModel codeViewModel = EditorWorkspace.This.ActiveDocument as CodeViewModel;
			CodeEditorMainWindow codeEditorMainWindow = codeViewModel.UI as CodeEditorMainWindow;
			if (codeEditorMainWindow == null)
			{
				return;
			}
			codeEditorMainWindow.Focus(addPointModel);
			base.DialogResult = new bool?(false);
		}

		// Token: 0x06000085 RID: 133 RVA: 0x00003C24 File Offset: 0x00001E24
		private void query_TextChanged(object sender, TextChangedEventArgs e)
		{
			if (string.IsNullOrEmpty(this.query.Text))
			{
				this.ResetList();
				return;
			}
			this.Filter(this.query.Text);
		}

		// Token: 0x06000086 RID: 134 RVA: 0x00003CA4 File Offset: 0x00001EA4
		private void Filter(string condition)
		{
			this._customerView = CollectionViewSource.GetDefaultView(this.dataGrid.ItemsSource);
			this._customerView.Filter = delegate(object item)
			{
				if (item is AddPointModel)
				{
					return ((AddPointModel)item).Name.Contains(condition);
				}
				return item is SectionModel && ((SectionModel)item).Name.Contains(condition);
			};
			this.dataGrid.SelectedIndex = 0;
		}

		// Token: 0x06000087 RID: 135 RVA: 0x00003CF7 File Offset: 0x00001EF7
		private void ResetList()
		{
			if (this._customerView != null)
			{
				this._customerView.Filter = null;
			}
			this.dataGrid.SelectedIndex = 0;
		}

		// Token: 0x06000088 RID: 136 RVA: 0x00003D19 File Offset: 0x00001F19
		public void Dispose()
		{
			base.DataContext = null;
		}

		// Token: 0x04000041 RID: 65
		private ICollectionView _customerView;

		// Token: 0x02000011 RID: 17
		public enum ContentType
		{
			// Token: 0x0400004A RID: 74
			src,
			// Token: 0x0400004B RID: 75
			cite,
			// Token: 0x0400004C RID: 76
			section
		}
	}
}
