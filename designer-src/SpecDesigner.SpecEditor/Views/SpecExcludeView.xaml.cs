using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000033 RID: 51
	public partial class SpecExcludeView : UserControl
	{
		// Token: 0x0600014C RID: 332 RVA: 0x0000A23F File Offset: 0x0000843F
		public SpecExcludeView(PackageKey programKey)
		{
			this.InitializeComponent();
			this.programKey = programKey;
			base.Loaded += this.SpecExcludeViewWindow_Loaded;
		}

		// Token: 0x0600014D RID: 333 RVA: 0x0000A266 File Offset: 0x00008466
		private void SpecExcludeViewWindow_Loaded(object sender, RoutedEventArgs e)
		{
			base.Loaded -= this.SpecExcludeViewWindow_Loaded;
			this.excludeDataGrid.DataContext = SettingManager.Get().GetTzpManger(this.programKey).SpecificationInfo.ExcludeNodes.ToList<string>();
		}

		// Token: 0x0600014E RID: 334 RVA: 0x0000A2A4 File Offset: 0x000084A4
		private void excludeDataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			if (sender != null)
			{
				DataGridRow dataGridRow = sender as DataGridRow;
				string text = dataGridRow.Item as string;
				SearchResultInfo searchResultInfo = new SearchResultInfo();
				searchResultInfo.Key = text;
				searchResultInfo.ProgramKey = this.programKey;
				searchResultInfo.SourceType = this.programKey.PackType;
				EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Publish(searchResultInfo);
			}
		}

		// Token: 0x0600014F RID: 335 RVA: 0x0000A304 File Offset: 0x00008504
		public static void ShowDialog(PackageKey key)
		{
			Window window = new Window
			{
				Title = (Application.Current.FindResource("specProperty_SpecFieldExcudeList") as string),
				Width = 300.0,
				Height = 300.0
			};
			window.Content = new SpecExcludeView(key);
			window.ShowDialog();
		}

		// Token: 0x06000152 RID: 338 RVA: 0x0000A3BC File Offset: 0x000085BC
		[DebuggerNonUserCode]
		[GeneratedCode("PresentationBuildTasks", "4.0.0.0")]
		[EditorBrowsable(EditorBrowsableState.Never)]
		void IStyleConnector.Connect(int connectionId, object target)
		{
			if (connectionId != 2)
			{
				return;
			}
			EventSetter eventSetter = new EventSetter();
			eventSetter.Event = Control.MouseDoubleClickEvent;
			eventSetter.Handler = new MouseButtonEventHandler(this.excludeDataGrid_MouseDoubleClick);
			((Style)target).Setters.Add(eventSetter);
		}

		// Token: 0x0400009D RID: 157
		private PackageKey programKey;
	}
}
