using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Media;
using System.Xml.Linq;
using Microsoft.Expression.Shapes;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Relation
{
	// Token: 0x0200005F RID: 95
	public partial class TBLControl : UserControl, IEntityControl, INotifyPropertyChanged
	{
		// Token: 0x060003A3 RID: 931 RVA: 0x00013E94 File Offset: 0x00012094
		public TBLControl(TBLModel tBLModel)
		{
			this.InitializeComponent();
			base.DataContext = tBLModel;
			base.LayoutUpdated += this.TBLControl_LayoutUpdated;
			this.ConnToChildArrow.MouseLeftButtonDown += this.ConnToChildArrow_MouseLeftButtonDown;
			this.ConnToParentArrow.MouseLeftButtonDown += this.ConnToParentArrow_MouseLeftButtonDown;
		}

		// Token: 0x1400000C RID: 12
		// (add) Token: 0x060003A4 RID: 932 RVA: 0x00013EF4 File Offset: 0x000120F4
		// (remove) Token: 0x060003A5 RID: 933 RVA: 0x00013F2C File Offset: 0x0001212C
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x170000A1 RID: 161
		// (get) Token: 0x060003A6 RID: 934 RVA: 0x00013F61 File Offset: 0x00012161
		public TBLModel Model
		{
			get
			{
				return base.DataContext as TBLModel;
			}
		}

		// Token: 0x170000A2 RID: 162
		// (get) Token: 0x060003A7 RID: 935 RVA: 0x00013F6E File Offset: 0x0001216E
		// (set) Token: 0x060003A8 RID: 936 RVA: 0x00013F78 File Offset: 0x00012178
		public TBLControl ParentTBL
		{
			get
			{
				return this._parentTBL;
			}
			set
			{
				this._parentTBL = value;
				if (this._parentTBL == null)
				{
					this.Model.Parent = "";
					this.Model.FkDetail = "";
					this.Model.FkMaster = "";
					return;
				}
				if (this.Model.Parent == value.Model.TBLName)
				{
					return;
				}
				this.Model.Parent = this._parentTBL.Model.TBLName;
				XElement fk = TableColumnHelper.GetFK(this.Model.Parent, this.Model.TBLName);
				if (fk != null)
				{
					this.Model.FkDetail = fk.Attribute("fk_detail").Value;
					this.Model.FkMaster = fk.Attribute("fk_master").Value;
				}
			}
		}

		// Token: 0x170000A3 RID: 163
		// (get) Token: 0x060003A9 RID: 937 RVA: 0x0001405D File Offset: 0x0001225D
		// (set) Token: 0x060003AA RID: 938 RVA: 0x00014065 File Offset: 0x00012265
		public RelationshipSpace Root { get; set; }

		// Token: 0x060003AB RID: 939 RVA: 0x0001406E File Offset: 0x0001226E
		public void AddChildren(TBLControl child)
		{
			child.ParentTBL = this;
			this.ChildrenContainer.Children.Add(child);
		}

		// Token: 0x060003AC RID: 940 RVA: 0x00014089 File Offset: 0x00012289
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x060003AD RID: 941 RVA: 0x000140A5 File Offset: 0x000122A5
		internal void ConnectDragOut()
		{
			this.IsConnectDragOver = false;
			this.Header.Background = ((!string.IsNullOrEmpty(this.Model.UpperTable)) ? Brushes.DodgerBlue : Brushes.White);
		}

		// Token: 0x060003AE RID: 942 RVA: 0x000140D7 File Offset: 0x000122D7
		internal void ConnectDragOver()
		{
			this.IsConnectDragOver = true;
			this.Header.Background = Brushes.DimGray;
		}

		// Token: 0x060003AF RID: 943 RVA: 0x000140F0 File Offset: 0x000122F0
		internal void RemoveChildren(TBLControl child)
		{
			this.ChildrenContainer.Children.Remove(child);
			child.ParentTBL = null;
		}

		// Token: 0x060003B0 RID: 944 RVA: 0x0001410A File Offset: 0x0001230A
		private void ConnToChildArrow_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			this.Root.StartConnect(this, this.ConnToChildArrow, ConnectionType.ConnectToChild);
		}

		// Token: 0x060003B1 RID: 945 RVA: 0x0001411F File Offset: 0x0001231F
		private void ConnToParentArrow_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			this.Root.StartConnect(this, this.ConnToParentArrow, ConnectionType.ConnectToParent);
		}

		// Token: 0x060003B2 RID: 946 RVA: 0x00014134 File Offset: 0x00012334
		private void CreateSubTBL_Click(object sender, RoutedEventArgs e)
		{
			this.Root.CreateSubTBLControl(this.Model.TBLName);
		}

		// Token: 0x060003B3 RID: 947 RVA: 0x0001414C File Offset: 0x0001234C
		private void DeleteImg_Click(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			string text = Application.Current.FindResource("ads_ActionDelConfirm") as string;
			if (DesignerMessageBox.Show(string.Format(text, this.Model.TBLName), "Delete", MessageBoxButton.YesNo) == MessageBoxResult.Yes)
			{
				IEnumerable<TBLControl> enumerable = this.ChildrenContainer.Children.OfType<TBLControl>().ToList<TBLControl>();
				foreach (TBLControl tblcontrol in enumerable)
				{
					this.RemoveChildren(tblcontrol);
					this.Root.AddChildren(tblcontrol);
				}
				this.isDelete = true;
				this.ChildrenContainer.Children.Clear();
				this.Root.RemoveTBLControl(this);
			}
		}

		// Token: 0x060003B4 RID: 948 RVA: 0x00014214 File Offset: 0x00012414
		private void AddChildTBLImg_Click(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			this.Root.CreateSubTBLControl(this.Model.TBLName);
		}

		// Token: 0x060003B5 RID: 949 RVA: 0x00014233 File Offset: 0x00012433
		private void Header_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			this.IsSelected = true;
		}

		// Token: 0x060003B6 RID: 950 RVA: 0x0001423C File Offset: 0x0001243C
		private void ModifyImg_Click(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			this.Root.ShowModifyModalBox(this.Model);
		}

		// Token: 0x060003B7 RID: 951 RVA: 0x00014258 File Offset: 0x00012458
		private void TBLControl_LayoutUpdated(object sender, EventArgs e)
		{
			if (this.isDelete)
			{
				return;
			}
			if (0.0 == base.RenderSize.Width)
			{
				return;
			}
			if (0.0 == base.RenderSize.Height)
			{
				return;
			}
			Size renderSize = this.Header.RenderSize;
			Point point = new Point(base.RenderSize.Width / 2.0, -2.0);
			Point point2 = new Point(base.RenderSize.Width / 2.0, renderSize.Height);
			this.HeaderAnchorPoint = base.TranslatePoint(point, this.Root.LayoutRoot);
			this.FootAnchorPoint = base.TranslatePoint(point2, this.Root.LayoutRoot);
			this.NotifyPropertyChanged("HeaderAnchorPoint");
			this.NotifyPropertyChanged("FootAnchorPoint");
		}

		// Token: 0x170000A4 RID: 164
		// (get) Token: 0x060003B8 RID: 952 RVA: 0x00014345 File Offset: 0x00012545
		// (set) Token: 0x060003B9 RID: 953 RVA: 0x00014357 File Offset: 0x00012557
		public Point HeaderAnchorPoint
		{
			get
			{
				return (Point)base.GetValue(TBLControl.HeaderAnchorPointProperty);
			}
			set
			{
				base.SetValue(TBLControl.HeaderAnchorPointProperty, value);
			}
		}

		// Token: 0x170000A5 RID: 165
		// (get) Token: 0x060003BA RID: 954 RVA: 0x0001436A File Offset: 0x0001256A
		// (set) Token: 0x060003BB RID: 955 RVA: 0x0001437C File Offset: 0x0001257C
		public Point FootAnchorPoint
		{
			get
			{
				return (Point)base.GetValue(TBLControl.FootAnchorPointProperty);
			}
			set
			{
				base.SetValue(TBLControl.FootAnchorPointProperty, value);
			}
		}

		// Token: 0x170000A6 RID: 166
		// (get) Token: 0x060003BC RID: 956 RVA: 0x0001438F File Offset: 0x0001258F
		// (set) Token: 0x060003BD RID: 957 RVA: 0x00014397 File Offset: 0x00012597
		public bool IsSelected
		{
			get
			{
				return this._isSelected;
			}
			set
			{
				this._isSelected = value;
				if (value)
				{
					this.Root.SelectedItem = this;
				}
				this.NotifyPropertyChanged("IsSelected");
			}
		}

		// Token: 0x060003BE RID: 958 RVA: 0x000143BC File Offset: 0x000125BC
		private void OnDeleteMenuItem_Click(object sender, RoutedEventArgs e)
		{
			TBLModel tblmodel = base.DataContext as TBLModel;
			if (tblmodel != null)
			{
				tblmodel.Status = "d";
			}
		}

		// Token: 0x040001E6 RID: 486
		public bool IsConnectDragOver;

		// Token: 0x040001E7 RID: 487
		public bool isDelete;

		// Token: 0x040001E9 RID: 489
		private TBLControl _parentTBL;

		// Token: 0x040001EA RID: 490
		public static readonly DependencyProperty HeaderAnchorPointProperty = DependencyProperty.Register("HeaderAnchorPoint", typeof(Point), typeof(TBLControl), new FrameworkPropertyMetadata(default(Point)));

		// Token: 0x040001EB RID: 491
		public static readonly DependencyProperty FootAnchorPointProperty = DependencyProperty.Register("FootAnchorPoint", typeof(Point), typeof(TBLControl), new FrameworkPropertyMetadata(default(Point)));

		// Token: 0x040001EC RID: 492
		private bool _isSelected;
	}
}
