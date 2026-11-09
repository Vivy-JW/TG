using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000043 RID: 67
	public partial class SearchKeySettingWindow : Window
	{
		// Token: 0x060002E9 RID: 745 RVA: 0x00018EBB File Offset: 0x000170BB
		public SearchKeySettingWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x060002EA RID: 746 RVA: 0x00018ED4 File Offset: 0x000170D4
		public SearchKeySettingWindow(PackageKey key)
			: this()
		{
			base.Title = Application.Current.FindResource("WinTitle_SearchKeySettingTitle") as string;
			this.ProgramKey = key;
			this._searchHistory = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).SearchHistory;
			this.AddKeyWord(SettingManager.Get().GetTzpManger(this.ProgramKey).infoXML);
			base.DataContext = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey);
		}

		// Token: 0x060002EB RID: 747 RVA: 0x00018F54 File Offset: 0x00017154
		private void Button_Click(object sender, RoutedEventArgs e)
		{
			if (string.IsNullOrEmpty(this.SearchCB.Text.Trim()))
			{
				MessageBox.Show(Application.Current.FindResource("Message_KeyWordNotNull") as string);
				return;
			}
			PackageKey packageKey = Application.Current.MainWindow.Tag as PackageKey;
			ResourceController.GetInstance().GetProgramInfo(packageKey).ClearNormalizationSearchResult();
			this.SearchingKey = this.SearchCB.Text;
			this.AddKeyWord(this.SearchCB.Text);
			string text = (this.SearchTargetCB.SelectedItem as ComboBoxItem).Tag.ToString();
			if (text.Equals("All Open Documents"))
			{
				using (Dictionary<PackageKey, TzpManager>.KeyCollection.Enumerator enumerator = SettingManager.Get().tzpMap.Keys.GetEnumerator())
				{
					while (enumerator.MoveNext())
					{
						PackageKey packageKey2 = enumerator.Current;
						if (SettingManager.Get().GetTzpManger(packageKey2).IsDiff)
						{
							SearchKeywordEventArgs e2 = new SearchKeywordEventArgs(packageKey2, this.SearchingKey, DiffType.Normal);
							EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e2);
							SearchKeywordEventArgs e3 = new SearchKeywordEventArgs(packageKey2, this.SearchingKey, DiffType.Diff);
							EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e3);
							SettingManager.Get().GetTzpManger(packageKey2).infoXML = this.SearchingKey;
						}
					}
					goto IL_0199;
				}
			}
			SearchKeywordEventArgs e4 = new SearchKeywordEventArgs(packageKey, this.SearchingKey, DiffType.Normal);
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e4);
			SearchKeywordEventArgs e5 = new SearchKeywordEventArgs(packageKey, this.SearchingKey, DiffType.Diff);
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e5);
			SettingManager.Get().GetTzpManger(packageKey).infoXML = this.SearchingKey;
			IL_0199:
			base.Close();
		}

		// Token: 0x060002EC RID: 748 RVA: 0x00019110 File Offset: 0x00017310
		internal void AddKeyWord(string searchKey)
		{
			if (!this._searchHistory.Contains(searchKey))
			{
				this._searchHistory.Add(searchKey);
				ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).SearchHistory = this._searchHistory;
			}
		}

		// Token: 0x04000146 RID: 326
		private PackageKey ProgramKey;

		// Token: 0x04000147 RID: 327
		private ObservableCollection<string> _searchHistory = new ObservableCollection<string>();

		// Token: 0x04000148 RID: 328
		private string SearchingKey;
	}
}
