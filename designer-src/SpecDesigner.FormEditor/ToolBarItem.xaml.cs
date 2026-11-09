using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Xml.Linq;

namespace SpecDesigner.FormEditor
{
	// Token: 0x02000048 RID: 72
	public partial class ToolBarItem : UserControl, INotifyPropertyChanged
	{
		// Token: 0x14000008 RID: 8
		// (add) Token: 0x0600029B RID: 667 RVA: 0x0000DB8C File Offset: 0x0000BD8C
		// (remove) Token: 0x0600029C RID: 668 RVA: 0x0000DBC4 File Offset: 0x0000BDC4
		public event ToolBarItem.ToolItemClick ToolItemClicked;

		// Token: 0x17000084 RID: 132
		// (get) Token: 0x0600029D RID: 669 RVA: 0x0000DBF9 File Offset: 0x0000BDF9
		// (set) Token: 0x0600029E RID: 670 RVA: 0x0000DC01 File Offset: 0x0000BE01
		public XElement Model { get; private set; }

		// Token: 0x0600029F RID: 671 RVA: 0x0000DC0A File Offset: 0x0000BE0A
		public ToolBarItem(XElement model)
		{
			this.InitializeComponent();
			this.widgetButton.Click += this.item_Click;
			this.Model = model;
		}

		// Token: 0x060002A0 RID: 672 RVA: 0x0000DC3D File Offset: 0x0000BE3D
		private void item_Click(object sender, RoutedEventArgs e)
		{
			if (this.ToolItemClicked != null && this.IsEnable)
			{
				this.ToolItemClicked(this, base.GetValue(ToolBarItem.ItemNameProperty).ToString());
			}
		}

		// Token: 0x17000085 RID: 133
		// (get) Token: 0x060002A1 RID: 673 RVA: 0x0000DC6B File Offset: 0x0000BE6B
		// (set) Token: 0x060002A2 RID: 674 RVA: 0x0000DC73 File Offset: 0x0000BE73
		public bool IsEnable
		{
			get
			{
				return this._isenable;
			}
			set
			{
				this._isenable = value;
				this.widgetButton.IsEnabled = value;
				this.widgetButton.Cursor = (value ? Cursors.Hand : null);
				this.NotifyPropertyChanged("IsEnable");
			}
		}

		// Token: 0x17000086 RID: 134
		// (get) Token: 0x060002A3 RID: 675 RVA: 0x0000DCA9 File Offset: 0x0000BEA9
		// (set) Token: 0x060002A4 RID: 676 RVA: 0x0000DCBB File Offset: 0x0000BEBB
		public string Text
		{
			get
			{
				return (string)base.GetValue(ToolBarItem.TextProperty);
			}
			set
			{
				base.SetValue(ToolBarItem.TextProperty, value);
			}
		}

		// Token: 0x17000087 RID: 135
		// (get) Token: 0x060002A5 RID: 677 RVA: 0x0000DCC9 File Offset: 0x0000BEC9
		// (set) Token: 0x060002A6 RID: 678 RVA: 0x0000DCDB File Offset: 0x0000BEDB
		public string ItemName
		{
			get
			{
				return (string)base.GetValue(ToolBarItem.ItemNameProperty);
			}
			set
			{
				base.SetValue(ToolBarItem.ItemNameProperty, value);
			}
		}

		// Token: 0x17000088 RID: 136
		// (get) Token: 0x060002A7 RID: 679 RVA: 0x0000DCE9 File Offset: 0x0000BEE9
		// (set) Token: 0x060002A8 RID: 680 RVA: 0x0000DCFB File Offset: 0x0000BEFB
		public string ToolImageSource
		{
			get
			{
				return base.GetValue(ToolBarItem.ToolImageSourceProperty).ToString();
			}
			set
			{
				base.SetValue(ToolBarItem.ToolImageSourceProperty, value);
			}
		}

		// Token: 0x14000009 RID: 9
		// (add) Token: 0x060002A9 RID: 681 RVA: 0x0000DD0C File Offset: 0x0000BF0C
		// (remove) Token: 0x060002AA RID: 682 RVA: 0x0000DD44 File Offset: 0x0000BF44
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060002AB RID: 683 RVA: 0x0000DD79 File Offset: 0x0000BF79
		public void NotifyPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x04000179 RID: 377
		private bool _isenable = true;

		// Token: 0x0400017A RID: 378
		public static readonly DependencyProperty TextProperty = DependencyProperty.Register("Text", typeof(string), typeof(ToolBarItem), null);

		// Token: 0x0400017B RID: 379
		public static readonly DependencyProperty ItemNameProperty = DependencyProperty.Register("ItemName", typeof(string), typeof(ToolBarItem), null);

		// Token: 0x0400017C RID: 380
		public static readonly DependencyProperty ToolImageSourceProperty = DependencyProperty.Register("ToolImageSource", typeof(string), typeof(ToolBarItem), new PropertyMetadata(null));

		// Token: 0x02000049 RID: 73
		// (Invoke) Token: 0x060002B0 RID: 688
		public delegate void ToolItemClick(object sender, string actionName);
	}
}
