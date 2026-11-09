using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Markup;
using System.Xml;

namespace SpecDesigner.FormDataEditor
{
	// Token: 0x02000006 RID: 6
	public partial class DetailView : UserControl
	{
		// Token: 0x17000004 RID: 4
		// (get) Token: 0x0600000D RID: 13 RVA: 0x000021A9 File Offset: 0x000003A9
		public bool HasItems
		{
			get
			{
				return this.detail != null && this.detail.HasItems;
			}
		}

		// Token: 0x0600000E RID: 14 RVA: 0x000021C0 File Offset: 0x000003C0
		public DetailView()
		{
			this.InitializeComponent();
			base.DataContextChanged += this.DetailView_DataContextChanged;
		}

		// Token: 0x0600000F RID: 15 RVA: 0x000021E0 File Offset: 0x000003E0
		private void DetailView_DataContextChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			this.detail.Columns.Clear();
			this.detail2.Columns.Clear();
			this.detail.Visibility = (this.detail2.Visibility = Visibility.Collapsed);
			base.Visibility = Visibility.Collapsed;
			if (!(base.DataContext is XmlNode))
			{
				return;
			}
			XmlNode xmlNode = base.DataContext as XmlNode;
			XmlNode xmlNode2 = (string.IsNullOrEmpty(xmlNode.InnerText) ? xmlNode.FirstChild : xmlNode.ChildNodes[1]);
			this.CreateColumns(xmlNode2, this.detail);
			if (xmlNode.LocalName == "zoom")
			{
				foreach (object obj in xmlNode.ChildNodes)
				{
					XmlNode xmlNode3 = (XmlNode)obj;
					if (xmlNode3.NodeType == XmlNodeType.Element && xmlNode3.LocalName != xmlNode2.LocalName)
					{
						this.CreateColumns(xmlNode3, this.detail2);
						break;
					}
				}
			}
			base.Visibility = this.detail.Visibility;
		}

		// Token: 0x06000010 RID: 16 RVA: 0x00002334 File Offset: 0x00000534
		private void CreateColumns(XmlNode node, DataGrid dataGrid)
		{
			if (node == null || node.Attributes == null)
			{
				return;
			}
			for (int i = 0; i < node.Attributes.Count; i++)
			{
				DataGridTextColumn dataGridTextColumn = new DataGridTextColumn();
				dataGridTextColumn.CanUserResize = true;
				dataGridTextColumn.Header = node.Attributes[i].Name;
				string text = string.Format("@{0}", node.Attributes[i].Name);
				dataGridTextColumn.Binding = new Binding
				{
					XPath = text
				};
				dataGrid.Columns.Add(dataGridTextColumn);
				dataGridTextColumn.Width = new DataGridLength(1.0, DataGridLengthUnitType.Auto);
			}
			IEnumerable<XmlNode> enumerable = from n in (base.DataContext as XmlNode).ChildNodes.OfType<XmlNode>()
				where n.LocalName == node.LocalName
				select n;
			dataGrid.DataContext = enumerable;
			dataGrid.Visibility = Visibility.Visible;
		}
	}
}
