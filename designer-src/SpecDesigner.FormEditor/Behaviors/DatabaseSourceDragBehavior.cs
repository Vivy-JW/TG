using System;
using System.Collections.ObjectModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Input;
using System.Windows.Interactivity;
using System.Xml.Linq;
using SpecDesigner.FormEditor.DBStructure;
using SpecDesigner.FormEditor.Helpers;
using SpecDesigner.FormEditor.Views;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x02000006 RID: 6
	public class DatabaseSourceDragBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x17000001 RID: 1
		// (get) Token: 0x0600001B RID: 27 RVA: 0x0000266F File Offset: 0x0000086F
		// (set) Token: 0x0600001C RID: 28 RVA: 0x00002677 File Offset: 0x00000877
		public FrameworkElement LayoutEditor { get; set; }

		// Token: 0x0600001D RID: 29 RVA: 0x00002680 File Offset: 0x00000880
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.MouseLeftButtonDown += this.AssociatedObject_MouseLeftButtonDown;
		}

		// Token: 0x0600001E RID: 30 RVA: 0x000026A0 File Offset: 0x000008A0
		private void AssociatedObject_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			Trace.WriteLine("AssociatedObject_MouseLeftButtonDown");
			XElement detail = (base.AssociatedObject.DataContext as DSColumnViewModel).Detail;
			string tblname = (base.AssociatedObject.DataContext as DSColumnViewModel).TBLName;
			Collection<PrepareAddColumn> collection = new Collection<PrepareAddColumn>();
			PrepareAddColumn prepareAddColumn;
			collection.Add(prepareAddColumn = new PrepareAddColumn
			{
				Table = tblname,
				Column = detail.Attribute("name").Value,
				Description = detail.Attribute("text").Value,
				Label = detail.Attribute("text").Value
			});
			XElement colField = TableColumnHelper.GetColField(prepareAddColumn.Table, prepareAddColumn.Column);
			if (colField != null)
			{
				prepareAddColumn.Widget = colField.Attribute("widget").Value;
				prepareAddColumn.Width = colField.Attribute("widget_width").Value;
			}
			ContainerType containerType = new ContainerType
			{
				Type = "None"
			};
			ContainerViewModel containerViewModel = new ContainerViewModel
			{
				Container = containerType
			};
			UICreator.Create(VisualTreeHelperEx.FindLogicVisualParent1<FormEditorMainWindow>(ManagedForm.Current), containerViewModel, collection, DatabaseSourceViewModel.This.ProgramKey);
		}
	}
}
