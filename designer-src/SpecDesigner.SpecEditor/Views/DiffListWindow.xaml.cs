using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000004 RID: 4
	public partial class DiffListWindow : Window
	{
		// Token: 0x17000003 RID: 3
		// (get) Token: 0x0600000B RID: 11 RVA: 0x00002170 File Offset: 0x00000370
		public static DiffListWindow This
		{
			get
			{
				if (DiffListWindow._this == null)
				{
					DiffListWindow._this = new DiffListWindow();
				}
				return DiffListWindow._this;
			}
		}

		// Token: 0x0600000C RID: 12 RVA: 0x00002188 File Offset: 0x00000388
		private DiffListWindow()
		{
			this.InitializeComponent();
			base.Title = Application.Current.FindResource("Diff_ListTitle") as string;
		}

		// Token: 0x0600000D RID: 13 RVA: 0x000021B0 File Offset: 0x000003B0
		private void Row_DoubleClick(object sender, MouseButtonEventArgs e)
		{
			DataGridRow dataGridRow = sender as DataGridRow;
			XElement xelement = dataGridRow.DataContext as XElement;
			if (xelement == null)
			{
				return;
			}
			string text = ((xelement.Attribute("name") != null) ? xelement.Attribute("name").Value : null);
			if (text == null)
			{
				return;
			}
			PackageKey packageKey = Application.Current.MainWindow.Tag as PackageKey;
			SearchResultInfo searchResultInfo = new SearchResultInfo
			{
				ProgramKey = packageKey,
				SourceType = TzpType.Form,
				Key = text
			};
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Publish(searchResultInfo);
		}

		// Token: 0x0600000E RID: 14 RVA: 0x0000224C File Offset: 0x0000044C
		public new void Show()
		{
			base.Show();
			if (base.Visibility != Visibility.Visible)
			{
				base.Visibility = Visibility.Visible;
			}
			base.Focus();
		}

		// Token: 0x0600000F RID: 15 RVA: 0x0000226A File Offset: 0x0000046A
		protected override void OnClosing(CancelEventArgs e)
		{
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
		}

		// Token: 0x06000012 RID: 18 RVA: 0x000022B8 File Offset: 0x000004B8
		[GeneratedCode("PresentationBuildTasks", "4.0.0.0")]
		[DebuggerNonUserCode]
		[EditorBrowsable(EditorBrowsableState.Never)]
		void IStyleConnector.Connect(int connectionId, object target)
		{
			if (connectionId != 1)
			{
				return;
			}
			EventSetter eventSetter = new EventSetter();
			eventSetter.Event = Control.MouseDoubleClickEvent;
			eventSetter.Handler = new MouseButtonEventHandler(this.Row_DoubleClick);
			((Style)target).Setters.Add(eventSetter);
		}

		// Token: 0x04000004 RID: 4
		private static DiffListWindow _this;
	}
}
