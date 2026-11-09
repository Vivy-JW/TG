using System;
using System.CodeDom.Compiler;
using System.Collections;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Media;
using System.Xml.Linq;
using Microsoft.Expression.Shapes;
using SpecDesigner.FormEditor.Helpers;
using SpecDesigner.FormEditor.Petzold.Media2D;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Relation
{
	// Token: 0x02000041 RID: 65
	public partial class RelationshipSpace : UserControl, INotifyPropertyChanged, IDisposable
	{
		// Token: 0x0600023B RID: 571 RVA: 0x0000BFAC File Offset: 0x0000A1AC
		public RelationshipSpace()
		{
			this.InitializeComponent();
			base.DataContext = this;
			this.programKey = Application.Current.MainWindow.Tag as PackageKey;
			this.createEntity(SettingManager.Get().GetTzpManger(this.programKey).SpecificationInfo.AssociateTable);
			this.createTableCBItems();
			base.CommandBindings.Add(new CommandBinding(RelationshipCommands.DeleteTBLControlCommand, new ExecutedRoutedEventHandler(this.OnExecutedDeleteTBLControl), new CanExecuteRoutedEventHandler(this.CanExecuteDeleteTBLControl)));
		}

		// Token: 0x0600023C RID: 572 RVA: 0x0000C057 File Offset: 0x0000A257
		private void CanExecuteDeleteTBLControl(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = null != this.SelectedItem;
		}

		// Token: 0x0600023D RID: 573 RVA: 0x0000C088 File Offset: 0x0000A288
		private void OnExecutedDeleteTBLControl(object sender, ExecutedRoutedEventArgs e)
		{
			string srName = e.Parameter.ToString();
			if (this.SelectedItem != null && !string.IsNullOrEmpty(srName))
			{
				TBLSRModel tblsrmodel = this.SelectedItem.Model.ScreenRecords.Where<TBLSRModel>((TBLSRModel record) => srName == record.SRName).ElementAtOrDefault<TBLSRModel>(0);
				if (tblsrmodel != null)
				{
					tblsrmodel.SetAttribute("status", "d");
				}
			}
		}

		// Token: 0x14000005 RID: 5
		// (add) Token: 0x0600023E RID: 574 RVA: 0x0000C104 File Offset: 0x0000A304
		// (remove) Token: 0x0600023F RID: 575 RVA: 0x0000C13C File Offset: 0x0000A33C
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x1700006F RID: 111
		// (get) Token: 0x06000240 RID: 576 RVA: 0x0000C171 File Offset: 0x0000A371
		// (set) Token: 0x06000241 RID: 577 RVA: 0x0000C179 File Offset: 0x0000A379
		public TBLControl SelectedItem
		{
			get
			{
				return this._selectedItem;
			}
			set
			{
				this._prevTbl = this._selectedItem;
				this._selectedItem = value;
				if (this._prevTbl != null && this._prevTbl != this._selectedItem)
				{
					this._prevTbl.IsSelected = false;
				}
			}
		}

		// Token: 0x17000070 RID: 112
		// (get) Token: 0x06000242 RID: 578 RVA: 0x0000C1C0 File Offset: 0x0000A3C0
		public IEnumerable<string> Tables
		{
			get
			{
				return from t in this._tableCBItems.Values
					where t.IsEnabled
					select t.Name;
			}
		}

		// Token: 0x06000243 RID: 579 RVA: 0x0000C21C File Offset: 0x0000A41C
		public void CreateNewTBLControl(string tblName, string parent)
		{
			TBLModel tblmodel = this._tableModel.CreateNewTBLModel(tblName, parent);
			if (this._tableCBItems.ContainsKey(tblName))
			{
				this._tableCBItems[tblName].IsEnabled = false;
			}
			TBLControl tblcontrol = new TBLControl(tblmodel)
			{
				Root = this
			};
			this.addToContainer(tblcontrol);
		}

		// Token: 0x06000244 RID: 580 RVA: 0x0000C26E File Offset: 0x0000A46E
		public void Dispose()
		{
			this._dragOverControl = null;
			this._dragStartControl = null;
			this._isConnecting = false;
			this._prevTbl = null;
			this._selectedItem = null;
			this._tableCBItems.Clear();
			this._tableModel.Dispose();
		}

		// Token: 0x06000245 RID: 581 RVA: 0x0000C2A9 File Offset: 0x0000A4A9
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x06000246 RID: 582 RVA: 0x0000C2EC File Offset: 0x0000A4EC
		public void RemoveTBLControl(TBLControl control)
		{
			TBLModel delModel = control.Model;
			this.RemoveRelation(delModel.TBLName);
			if (this._tableCBItems.ContainsKey(delModel.TBLName))
			{
				this._tableCBItems[delModel.TBLName].IsEnabled = true;
			}
			TBLControl tblcontrol = this._entities.Where<TBLControl>((TBLControl t) => t.Model.TBLName == delModel.Parent).FirstOrDefault<TBLControl>();
			this._entities.Remove(control);
			if (tblcontrol != null)
			{
				tblcontrol.RemoveChildren(control);
			}
			else
			{
				this.Container.Children.Remove(control);
			}
			this._tableModel.Remove(delModel);
		}

		// Token: 0x06000247 RID: 583 RVA: 0x0000C3A8 File Offset: 0x0000A5A8
		internal void AddChildren(TBLControl child)
		{
			this.RemoveRelation(child.Model.TBLName);
			this.Container.Children.Add(child);
		}

		// Token: 0x06000248 RID: 584 RVA: 0x0000C3D0 File Offset: 0x0000A5D0
		internal void CreateSubTBLControl(string parent)
		{
			this.createTableCBItems();
			this.CreateTBLBox.DataContext = this._tableModel.CreateNewTBLModel("", parent);
			this.CreateTBLBox.Visibility = Visibility.Visible;
			this.Canvas1.Visibility = Visibility.Visible;
			this._moveType = RelationshipSpace.MoveType.Create;
		}

		// Token: 0x06000249 RID: 585 RVA: 0x0000C420 File Offset: 0x0000A620
		internal void ShowModifyModalBox(TBLModel tblModel)
		{
			this._modifiedModel = tblModel;
			this.createTableCBItems(this._modifiedModel);
			this._tempTBLModel = RelationshipSpace.TBLModelForCheck.Create(tblModel);
			this.UpdateUpperTableList(tblModel);
			this.ModifyTBLBox.DataContext = this._tempTBLModel;
			this.ModifyTBLBox.Visibility = Visibility.Visible;
			this.Canvas1.Visibility = Visibility.Visible;
			this._moveType = RelationshipSpace.MoveType.Modify;
		}

		// Token: 0x17000071 RID: 113
		// (get) Token: 0x0600024A RID: 586 RVA: 0x0000C494 File Offset: 0x0000A694
		public IEnumerable<string> UpperTables
		{
			get
			{
				if (this._upperTables == null)
				{
					return null;
				}
				return from t in this._upperTables
					where t.IsEnabled
					select t.Name;
			}
		}

		// Token: 0x0600024B RID: 587 RVA: 0x0000C4F8 File Offset: 0x0000A6F8
		private void UpdateUpperTableList(TBLModel tBLModel)
		{
			if (this._upperTables == null)
			{
				this._upperTables = new List<TableComboBoxItem>();
			}
			else
			{
				this._upperTables.Clear();
			}
			this._upperTables.Add(new TableComboBoxItem
			{
				Name = "",
				Text = "------",
				IsEnabled = true
			});
			TableAssociationModel associateTable = SettingManager.Get().GetTzpManger(tBLModel.Key).SpecificationInfo.AssociateTable;
			foreach (TBLModel tblmodel in associateTable.AliveTBLs)
			{
				this._upperTables.Add(new TableComboBoxItem
				{
					Name = tblmodel.TBLName,
					Text = string.Format("{0} {1}", tblmodel.TBLName, TableColumnHelper.GetTableDesc(tblmodel.TBLName)),
					IsEnabled = (tblmodel != tBLModel && tblmodel.UpperTable != tBLModel.TBLName)
				});
			}
			this.NotifyPropertyChanged("UpperTables");
		}

		// Token: 0x0600024C RID: 588 RVA: 0x0000C618 File Offset: 0x0000A818
		private void CancelTBLBox_MouseLeftButtonUp(object sender, RoutedEventArgs e)
		{
			e.Handled = true;
			this.ModifyTBLBox.Visibility = Visibility.Collapsed;
			this.Canvas1.Visibility = Visibility.Collapsed;
			this._moveType = RelationshipSpace.MoveType.Null;
			this.ModifyTBLBox.DataContext = Binding.DoNothing;
		}

		// Token: 0x0600024D RID: 589 RVA: 0x0000C650 File Offset: 0x0000A850
		private void modifyUpperTBLNameCB_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			TBLModel tblmodel = base.DataContext as TBLModel;
			if (tblmodel == null)
			{
				return;
			}
			string text = this.modifyUpperTBLNameCB.SelectedValue as string;
			if (!string.IsNullOrEmpty(text))
			{
				this.modifyUpperKeyTB.Text = TableColumnHelper.GetPK(text);
				if (string.IsNullOrEmpty(this.modifyThisKeyTB.Text))
				{
					this.modifyThisKeyTB.Text = TableColumnHelper.GetPK(tblmodel.TBLName);
				}
			}
			else
			{
				this.modifyUpperKeyTB.Text = (this.modifyThisKeyTB.Text = string.Empty);
			}
			this.modifyUpperKeyTB.IsEnabled = (this.modifyThisKeyTB.IsEnabled = !string.IsNullOrEmpty(text));
		}

		// Token: 0x17000072 RID: 114
		// (get) Token: 0x0600024E RID: 590 RVA: 0x0000C711 File Offset: 0x0000A911
		public RelayCommand ConfirmCommand
		{
			get
			{
				if (this._confirmCommand == null)
				{
					this._confirmCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteConfirm();
					}, (object p) => this.CanExecuteConfirm());
				}
				return this._confirmCommand;
			}
		}

		// Token: 0x0600024F RID: 591 RVA: 0x0000C744 File Offset: 0x0000A944
		private bool CanExecuteConfirm()
		{
			return RelationshipSpace.IsValid(this);
		}

		// Token: 0x06000250 RID: 592 RVA: 0x0000C74C File Offset: 0x0000A94C
		private void ExecuteConfirm()
		{
			this._modifiedModel.TBLName = this._tempTBLModel.TBLName;
			this._modifiedModel.PK = this._tempTBLModel.PK;
			this._modifiedModel.FkDetail = this._tempTBLModel.FkDetail;
			this._modifiedModel.FkMaster = this._tempTBLModel.FkMaster;
			this._modifiedModel.UpperKey = this._tempTBLModel.UpperKey;
			this._modifiedModel.UpperTable = this._tempTBLModel.UpperTable;
			this._modifiedModel.ThisKey = this._tempTBLModel.ThisKey;
			this.ModifyTBLBox.Visibility = Visibility.Collapsed;
			this.Canvas1.Visibility = Visibility.Collapsed;
			this._moveType = RelationshipSpace.MoveType.Null;
			this.ModifyTBLBox.DataContext = Binding.DoNothing;
			this._modifiedModel = null;
			this._tempTBLModel = null;
		}

		// Token: 0x06000251 RID: 593 RVA: 0x0000C830 File Offset: 0x0000AA30
		public static bool IsValid(DependencyObject parent)
		{
			bool flag = true;
			LocalValueEnumerator localValueEnumerator = parent.GetLocalValueEnumerator();
			while (localValueEnumerator.MoveNext())
			{
				LocalValueEntry localValueEntry = localValueEnumerator.Current;
				if (BindingOperations.IsDataBound(parent, localValueEntry.Property))
				{
					BindingOperations.GetBinding(parent, localValueEntry.Property);
					BindingExpression bindingExpression = BindingOperations.GetBindingExpression(parent, localValueEntry.Property);
					bindingExpression.UpdateSource();
					if (bindingExpression.HasError)
					{
						flag = false;
					}
				}
			}
			IEnumerable children = LogicalTreeHelper.GetChildren(parent);
			foreach (object obj in children)
			{
				if (obj is DependencyObject)
				{
					DependencyObject dependencyObject = (DependencyObject)obj;
					if (!RelationshipSpace.IsValid(dependencyObject))
					{
						flag = false;
					}
				}
			}
			return flag;
		}

		// Token: 0x06000252 RID: 594 RVA: 0x0000C900 File Offset: 0x0000AB00
		internal void StartConnect(TBLControl tBLControl, BlockArrow ConnArrow, ConnectionType type)
		{
			this._connectType = type;
			this.createDragConnector(ConnArrow);
			this._dragStartControl = tBLControl;
			base.CaptureMouse();
			base.MouseMove -= this.RelationshipSpace_MouseMove;
			base.MouseMove += this.RelationshipSpace_MouseMove;
			base.MouseLeftButtonUp -= this.RelationshipSpace_MouseLeftButtonUp;
			base.MouseLeftButtonUp += this.RelationshipSpace_MouseLeftButtonUp;
			this._isConnecting = true;
		}

		// Token: 0x06000253 RID: 595 RVA: 0x0000C978 File Offset: 0x0000AB78
		private void AcceptNewTBLButton_Click(object sender, RoutedEventArgs e)
		{
			TBLModel tblmodel = this.CreateTBLBox.DataContext as TBLModel;
			if (tblmodel != null)
			{
				this.CreateNewTBLControl(tblmodel);
			}
			this.CreateTBLBox.DataContext = Binding.DoNothing;
			this.CreateTBLBox.Visibility = Visibility.Collapsed;
			this.Canvas1.Visibility = Visibility.Collapsed;
			this._moveType = RelationshipSpace.MoveType.Null;
		}

		// Token: 0x06000254 RID: 596 RVA: 0x0000C9D0 File Offset: 0x0000ABD0
		private void CancelNewTBLButton_Click(object sender, RoutedEventArgs e)
		{
			TBLModel tblmodel = this.CreateTBLBox.DataContext as TBLModel;
			if (tblmodel != null)
			{
				tblmodel.DeleteFromPersistence();
			}
			this.CreateTBLBox.DataContext = Binding.DoNothing;
			this.CreateTBLBox.Visibility = Visibility.Collapsed;
			this.Canvas1.Visibility = Visibility.Collapsed;
			this._moveType = RelationshipSpace.MoveType.Null;
		}

		// Token: 0x06000255 RID: 597 RVA: 0x0000CA28 File Offset: 0x0000AC28
		private void createConnector(TBLControl src, TBLControl dest)
		{
			TBLRelation tblrelation = new TBLRelation();
			tblrelation.SourceController = src;
			tblrelation.TargetController = dest;
			tblrelation.Tag = dest.Model.TBLName;
			this.LayoutRoot.Children.Insert(0, tblrelation);
		}

		// Token: 0x06000256 RID: 598 RVA: 0x0000CA6C File Offset: 0x0000AC6C
		private void createDragConnector(BlockArrow arrow)
		{
			double num = ((this._connectType == ConnectionType.ConnectToChild) ? arrow.Height : 0.0);
			Point point = arrow.TransformToVisual(this.LayoutRoot).Transform(new Point(arrow.Width / 2.0, num));
			this._connectingLine = new ArrowLine();
			this._connectingLine.Stroke = ((this._connectType == ConnectionType.ConnectToChild) ? (base.FindResource("ConnectionToChildBrush") as Brush) : (base.FindResource("ConnectionToParentBrush") as Brush));
			this._connectingLine.StrokeThickness = 3.0;
			this._connectingLine.X1 = point.X;
			this._connectingLine.Y1 = point.Y;
			this._connectingLine.X2 = point.X;
			this._connectingLine.Y2 = point.Y;
			this.LayoutRoot.Children.Add(this._connectingLine);
		}

		// Token: 0x06000257 RID: 599 RVA: 0x0000CB70 File Offset: 0x0000AD70
		private void createEntity(TableAssociationModel tableElement)
		{
			this._tableModel = tableElement;
			Dictionary<string, TBLControl> dictionary = new Dictionary<string, TBLControl>();
			if (this._entities.Count > 0)
			{
				this._entities.Clear();
			}
			foreach (TBLModel tblmodel in this._tableModel.AliveTBLs)
			{
				if (!(tblmodel.Status == ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)))
				{
					TBLControl tblcontrol = new TBLControl(tblmodel)
					{
						Root = this
					};
					if (tblmodel.Parent == "")
					{
						this.Container.Children.Add(tblcontrol);
					}
					if (!dictionary.ContainsKey(tblmodel.TBLName))
					{
						dictionary.Add(tblmodel.TBLName, tblcontrol);
					}
					this._entities.Add(tblcontrol);
				}
			}
			foreach (TBLControl tblcontrol2 in this._entities)
			{
				if (tblcontrol2.Model.Parent != "")
				{
					TBLControl tblcontrol3 = dictionary[tblcontrol2.Model.Parent];
					this.createConnector(tblcontrol3, tblcontrol2);
					tblcontrol3.AddChildren(tblcontrol2);
				}
			}
			dictionary.Clear();
		}

		// Token: 0x06000258 RID: 600 RVA: 0x0000CCE0 File Offset: 0x0000AEE0
		private void CreateNewTBLControl(TBLModel newModel)
		{
			string tblname = newModel.TBLName;
			if (tblname.Length == 0)
			{
				newModel.DeleteFromPersistence();
				return;
			}
			string parent = newModel.Parent;
			if (this._tableCBItems.ContainsKey(tblname))
			{
				this._tableCBItems[tblname].IsEnabled = false;
			}
			TBLControl tblcontrol = new TBLControl(newModel)
			{
				Root = this
			};
			this.addToContainer(tblcontrol);
		}

		// Token: 0x06000259 RID: 601 RVA: 0x0000CD6C File Offset: 0x0000AF6C
		private void addToContainer(TBLControl control)
		{
			this._entities.Add(control);
			if (string.IsNullOrEmpty(control.Model.Parent))
			{
				this.Container.Children.Add(control);
				return;
			}
			TBLControl tblcontrol = this._entities.Where<TBLControl>((TBLControl t) => t.Model.TBLName == control.Model.Parent).FirstOrDefault<TBLControl>();
			if (tblcontrol != null)
			{
				control.ParentTBL = tblcontrol;
				tblcontrol.AddChildren(control);
				this.createConnector(tblcontrol, control);
				return;
			}
			this.Container.Children.Add(control);
		}

		// Token: 0x0600025A RID: 602 RVA: 0x0000CE2C File Offset: 0x0000B02C
		private void CreateNewTBLMenuItem_Click(object sender, RoutedEventArgs e)
		{
			this.createTableCBItems();
			this.CreateTBLBox.DataContext = this._tableModel.CreateNewTBLModel("", "");
			this.CreateTBLBox.Visibility = Visibility.Visible;
			this._moveType = RelationshipSpace.MoveType.Create;
			this.Canvas1.Visibility = Visibility.Visible;
		}

		// Token: 0x0600025B RID: 603 RVA: 0x0000CE7E File Offset: 0x0000B07E
		private void createTableCBItems()
		{
			this.createTableCBItems(null);
		}

		// Token: 0x0600025C RID: 604 RVA: 0x0000CE88 File Offset: 0x0000B088
		private void createTableCBItems(TBLModel tblModel)
		{
			IEnumerable<XElement> tables = TableColumnHelper.GetTables();
			if (this._tableCBItems.Count == 0)
			{
				foreach (XElement xelement in tables)
				{
					this._tableCBItems.Add(xelement.Attribute("name").Value, new TableComboBoxItem
					{
						Name = xelement.Attribute("name").Value,
						Text = string.Format("{0}  {1}", xelement.Attribute("name").Value, xelement.Attribute("desc").Value),
						IsEnabled = true
					});
				}
			}
			foreach (TBLControl tblcontrol in this._entities)
			{
				tblcontrol.IsEnabled = true;
				TableComboBoxItem tableComboBoxItem = null;
				if (this._tableCBItems.TryGetValue(tblcontrol.Model.TBLName, out tableComboBoxItem))
				{
					tableComboBoxItem.IsEnabled = tblModel != null && tblModel.TBLName == tableComboBoxItem.Name;
				}
			}
			this.NotifyPropertyChanged("Tables");
		}

		// Token: 0x17000073 RID: 115
		// (get) Token: 0x0600025D RID: 605 RVA: 0x0000CFF8 File Offset: 0x0000B1F8
		public ObservableCollection<TBLControl> Entities
		{
			get
			{
				return this._entities;
			}
		}

		// Token: 0x0600025E RID: 606 RVA: 0x0000D000 File Offset: 0x0000B200
		private static T GetVisualChild<T>(DependencyObject parent) where T : Visual
		{
			T t = default(T);
			int childrenCount = VisualTreeHelper.GetChildrenCount(parent);
			for (int i = 0; i < childrenCount; i++)
			{
				Visual visual = (Visual)VisualTreeHelper.GetChild(parent, i);
				t = visual as T;
				if (t == null)
				{
					t = RelationshipSpace.GetVisualChild<T>(visual);
				}
				if (t != null)
				{
					break;
				}
			}
			return t;
		}

		// Token: 0x0600025F RID: 607 RVA: 0x0000D05C File Offset: 0x0000B25C
		private Panel GetItemsPanel(DependencyObject itemsControl)
		{
			ItemsPresenter visualChild = RelationshipSpace.GetVisualChild<ItemsPresenter>(itemsControl);
			return VisualTreeHelper.GetChild(visualChild, 0) as Panel;
		}

		// Token: 0x06000260 RID: 608 RVA: 0x0000D080 File Offset: 0x0000B280
		private void MoveTBL(TBLControl parent, TBLControl child)
		{
			this.RemoveRelation(child.Model.TBLName);
			if (child.ParentTBL == null)
			{
				this.Container.Children.Remove(child);
			}
			else
			{
				child.ParentTBL.RemoveChildren(child);
			}
			parent.AddChildren(child);
			this.createConnector(parent, child);
		}

		// Token: 0x06000261 RID: 609 RVA: 0x0000D0D4 File Offset: 0x0000B2D4
		private void RelationshipSpace_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			if (this._dragStartControl != null && this._dragOverControl != null)
			{
				switch (this._connectType)
				{
				case ConnectionType.ConnectToParent:
					this.MoveTBL(this._dragOverControl, this._dragStartControl);
					break;
				case ConnectionType.ConnectToChild:
					this.MoveTBL(this._dragStartControl, this._dragOverControl);
					break;
				}
			}
			base.MouseLeftButtonUp -= this.RelationshipSpace_MouseLeftButtonUp;
			if (this._dragOverControl != null)
			{
				this._dragOverControl.ConnectDragOut();
			}
			this._dragOverControl = null;
			this._dragStartControl = null;
			this._connectType = ConnectionType.None;
			this.LayoutRoot.Children.Remove(this._connectingLine);
			this._connectingLine = null;
			this._isConnecting = false;
			base.ReleaseMouseCapture();
		}

		// Token: 0x06000262 RID: 610 RVA: 0x0000D194 File Offset: 0x0000B394
		private void RelationshipSpace_MouseMove(object sender, MouseEventArgs e)
		{
			if (this._isConnecting)
			{
				Point position = e.GetPosition(this.LayoutRoot);
				if ((int)position.X % 2 != 0 && (int)position.Y % 2 != 0)
				{
					return;
				}
				if (this._connectingLine != null)
				{
					this._connectingLine.X2 = position.X + 2.0;
					this._connectingLine.Y2 = position.Y + 2.0;
					HitTestResult hitTestResult = VisualTreeHelper.HitTest(this.Container, position);
					if (hitTestResult != null)
					{
						TBLControl tblcontrol = VisualTreeHelperEx.FindVisualParent1<TBLControl>(hitTestResult.VisualHit);
						if (tblcontrol == this._dragStartControl)
						{
							tblcontrol = null;
							this._dragOverControl = null;
						}
						else
						{
							switch (this._connectType)
							{
							case ConnectionType.ConnectToParent:
							{
								string text = this._dragStartControl.Model.TBLName;
								for (TBLControl tblcontrol2 = tblcontrol; tblcontrol2 != null; tblcontrol2 = tblcontrol2.ParentTBL)
								{
									if (tblcontrol2.Model.TBLName == text)
									{
										tblcontrol = null;
										break;
									}
								}
								break;
							}
							case ConnectionType.ConnectToChild:
							{
								string text = tblcontrol.Model.TBLName;
								for (TBLControl tblcontrol2 = this._dragStartControl; tblcontrol2 != null; tblcontrol2 = tblcontrol2.ParentTBL)
								{
									if (tblcontrol2.Model.TBLName == text)
									{
										tblcontrol = null;
										break;
									}
								}
								break;
							}
							}
						}
						if (this._dragOverControl != tblcontrol)
						{
							if (tblcontrol != null)
							{
								if (this._dragOverControl != tblcontrol && this._dragOverControl != null)
								{
									this._dragOverControl.ConnectDragOut();
								}
								tblcontrol.ConnectDragOver();
							}
							else
							{
								this._dragOverControl = null;
							}
						}
						else if (tblcontrol == null && this._dragOverControl != null)
						{
							this._dragOverControl.ConnectDragOut();
						}
						this._dragOverControl = tblcontrol;
						return;
					}
					if (this._dragOverControl != null)
					{
						this._dragOverControl.ConnectDragOut();
					}
					this._dragOverControl = null;
				}
			}
		}

		// Token: 0x06000263 RID: 611 RVA: 0x0000D36C File Offset: 0x0000B56C
		private void RemoveRelation(string tblName)
		{
			TBLRelation tblrelation = (from c in this.LayoutRoot.Children.OfType<TBLRelation>()
				where tblName.Equals(c.Tag)
				select c).FirstOrDefault<TBLRelation>();
			if (tblrelation != null)
			{
				this.LayoutRoot.Children.Remove(tblrelation);
				tblrelation.SourceController = null;
				tblrelation.TargetController = null;
				tblrelation.Tag = null;
			}
		}

		// Token: 0x06000264 RID: 612 RVA: 0x0000D3D8 File Offset: 0x0000B5D8
		private void Canvas1_MouseMove(object sender, MouseEventArgs e)
		{
			UIElement uielement = Mouse.Captured as UIElement;
			if (e.LeftButton == MouseButtonState.Pressed && uielement != null)
			{
				Point position = e.GetPosition(this.Canvas1);
				Canvas.SetLeft(uielement, position.X - this.targetPoint.X);
				Canvas.SetTop(uielement, position.Y - this.targetPoint.Y);
			}
		}

		// Token: 0x06000265 RID: 613 RVA: 0x0000D43C File Offset: 0x0000B63C
		private void Canvas1_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			switch (this._moveType)
			{
			case RelationshipSpace.MoveType.Modify:
				this.ModifyTBLBox.PointToScreen(this.targetPoint);
				Mouse.Capture(null);
				return;
			case RelationshipSpace.MoveType.Create:
				this.CreateTBLBox.PointToScreen(this.targetPoint);
				Mouse.Capture(null);
				return;
			default:
				return;
			}
		}

		// Token: 0x06000266 RID: 614 RVA: 0x0000D494 File Offset: 0x0000B694
		private void Canvas1_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			switch (this._moveType)
			{
			case RelationshipSpace.MoveType.Modify:
				this.targetPoint = e.GetPosition(this.ModifyTBLBox);
				this.ModifyTBLBox.CaptureMouse();
				return;
			case RelationshipSpace.MoveType.Create:
				this.targetPoint = e.GetPosition(this.CreateTBLBox);
				this.CreateTBLBox.CaptureMouse();
				return;
			default:
				return;
			}
		}

		// Token: 0x0400013D RID: 317
		public TBLControl _dragOverControl;

		// Token: 0x0400013E RID: 318
		private ArrowLine _connectingLine;

		// Token: 0x0400013F RID: 319
		private ConnectionType _connectType;

		// Token: 0x04000140 RID: 320
		private TBLControl _dragStartControl;

		// Token: 0x04000141 RID: 321
		private bool _isConnecting;

		// Token: 0x04000142 RID: 322
		private TBLControl _prevTbl;

		// Token: 0x04000143 RID: 323
		private TBLControl _selectedItem;

		// Token: 0x04000144 RID: 324
		private Dictionary<string, TableComboBoxItem> _tableCBItems = new Dictionary<string, TableComboBoxItem>();

		// Token: 0x04000145 RID: 325
		private TableAssociationModel _tableModel;

		// Token: 0x04000146 RID: 326
		private PackageKey programKey;

		// Token: 0x04000148 RID: 328
		private RelationshipSpace.TBLModelForCheck _tempTBLModel;

		// Token: 0x04000149 RID: 329
		private TBLModel _modifiedModel;

		// Token: 0x0400014A RID: 330
		private List<TableComboBoxItem> _upperTables;

		// Token: 0x0400014B RID: 331
		private RelayCommand _confirmCommand;

		// Token: 0x0400014C RID: 332
		public ObservableCollection<TBLControl> _entities = new ObservableCollection<TBLControl>();

		// Token: 0x0400014D RID: 333
		private Point targetPoint;

		// Token: 0x0400014E RID: 334
		private RelationshipSpace.MoveType _moveType = RelationshipSpace.MoveType.Null;

		// Token: 0x02000042 RID: 66
		private class TBLModelForCheck : INotifyPropertyChanged, IDataErrorInfo
		{
			// Token: 0x0600026F RID: 623 RVA: 0x0000D720 File Offset: 0x0000B920
			public static RelationshipSpace.TBLModelForCheck Create(TBLModel model)
			{
				return new RelationshipSpace.TBLModelForCheck
				{
					PK = model.PK,
					_parent = model.Parent,
					TBLName = model.TBLName,
					_fkMaster = model.FkMaster,
					_fkDetail = model.FkDetail,
					UpperKey = model.UpperKey,
					UpperTable = model.UpperTable,
					ThisKey = model.ThisKey,
					Main = model.Main
				};
			}

			// Token: 0x17000074 RID: 116
			// (get) Token: 0x06000271 RID: 625 RVA: 0x0000D7B3 File Offset: 0x0000B9B3
			// (set) Token: 0x06000272 RID: 626 RVA: 0x0000D7BB File Offset: 0x0000B9BB
			public string TBLName
			{
				get
				{
					return this._tblName;
				}
				set
				{
					this._tblName = value;
					this.OnPropertyChanged("TBLName");
					this.OnPropertyChanged("TBLDesc");
				}
			}

			// Token: 0x17000075 RID: 117
			// (get) Token: 0x06000273 RID: 627 RVA: 0x0000D7DA File Offset: 0x0000B9DA
			public string TBLDesc
			{
				get
				{
					if (!string.IsNullOrEmpty(this.TBLName))
					{
						return TableColumnHelper.GetTableDesc(this.TBLName);
					}
					return string.Empty;
				}
			}

			// Token: 0x17000076 RID: 118
			// (get) Token: 0x06000274 RID: 628 RVA: 0x0000D7FA File Offset: 0x0000B9FA
			// (set) Token: 0x06000275 RID: 629 RVA: 0x0000D802 File Offset: 0x0000BA02
			public string PK
			{
				get
				{
					return this._pk;
				}
				set
				{
					this._pk = value;
					this.OnPropertyChanged("PK");
				}
			}

			// Token: 0x17000077 RID: 119
			// (get) Token: 0x06000276 RID: 630 RVA: 0x0000D816 File Offset: 0x0000BA16
			// (set) Token: 0x06000277 RID: 631 RVA: 0x0000D81E File Offset: 0x0000BA1E
			public string Parent
			{
				get
				{
					return this._parent;
				}
				set
				{
					this._parent = value;
					this.OnPropertyChanged("Parent");
				}
			}

			// Token: 0x17000078 RID: 120
			// (get) Token: 0x06000278 RID: 632 RVA: 0x0000D832 File Offset: 0x0000BA32
			// (set) Token: 0x06000279 RID: 633 RVA: 0x0000D83A File Offset: 0x0000BA3A
			public string FkMaster
			{
				get
				{
					return this._fkMaster;
				}
				set
				{
					if (Regex.IsMatch(value, RelationshipSpace.TBLModelForCheck.ColumnCheckPattern))
					{
						throw new Exception("不允許常數");
					}
					this._fkMaster = value;
					this.OnPropertyChanged("FKMaster");
				}
			}

			// Token: 0x17000079 RID: 121
			// (get) Token: 0x0600027A RID: 634 RVA: 0x0000D866 File Offset: 0x0000BA66
			// (set) Token: 0x0600027B RID: 635 RVA: 0x0000D86E File Offset: 0x0000BA6E
			public string FkDetail
			{
				get
				{
					return this._fkDetail;
				}
				set
				{
					if (Regex.IsMatch(value, RelationshipSpace.TBLModelForCheck.ColumnCheckPattern))
					{
						throw new Exception("不允許常數");
					}
					this._fkDetail = value;
					this.OnPropertyChanged("FkDetail");
				}
			}

			// Token: 0x1700007A RID: 122
			// (get) Token: 0x0600027C RID: 636 RVA: 0x0000D89A File Offset: 0x0000BA9A
			// (set) Token: 0x0600027D RID: 637 RVA: 0x0000D8A2 File Offset: 0x0000BAA2
			public string UpperTable
			{
				get
				{
					return this._upperTable;
				}
				set
				{
					this._upperTable = value;
					this.OnPropertyChanged("UpperTable");
					this.OnPropertyChanged("UpperTableDesc");
				}
			}

			// Token: 0x1700007B RID: 123
			// (get) Token: 0x0600027E RID: 638 RVA: 0x0000D8C1 File Offset: 0x0000BAC1
			public string UpperTableDesc
			{
				get
				{
					if (!string.IsNullOrEmpty(this.UpperTable))
					{
						return TableColumnHelper.GetTableDesc(this.UpperTable);
					}
					return string.Empty;
				}
			}

			// Token: 0x1700007C RID: 124
			// (get) Token: 0x0600027F RID: 639 RVA: 0x0000D8E1 File Offset: 0x0000BAE1
			// (set) Token: 0x06000280 RID: 640 RVA: 0x0000D8E9 File Offset: 0x0000BAE9
			public string UpperKey
			{
				get
				{
					return this._upperKey;
				}
				set
				{
					this._upperKey = value;
					this.OnPropertyChanged("UpperKey");
				}
			}

			// Token: 0x1700007D RID: 125
			// (get) Token: 0x06000281 RID: 641 RVA: 0x0000D8FD File Offset: 0x0000BAFD
			// (set) Token: 0x06000282 RID: 642 RVA: 0x0000D905 File Offset: 0x0000BB05
			public string ThisKey
			{
				get
				{
					return this._thisKey;
				}
				set
				{
					this._thisKey = value;
					this.OnPropertyChanged("ThisKey");
				}
			}

			// Token: 0x1700007E RID: 126
			// (get) Token: 0x06000283 RID: 643 RVA: 0x0000D919 File Offset: 0x0000BB19
			// (set) Token: 0x06000284 RID: 644 RVA: 0x0000D921 File Offset: 0x0000BB21
			public string Main { get; set; }

			// Token: 0x14000006 RID: 6
			// (add) Token: 0x06000285 RID: 645 RVA: 0x0000D92C File Offset: 0x0000BB2C
			// (remove) Token: 0x06000286 RID: 646 RVA: 0x0000D964 File Offset: 0x0000BB64
			public event PropertyChangedEventHandler PropertyChanged;

			// Token: 0x06000287 RID: 647 RVA: 0x0000D999 File Offset: 0x0000BB99
			private void OnPropertyChanged(string propertyName)
			{
				if (this.PropertyChanged != null)
				{
					this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
				}
			}

			// Token: 0x1700007F RID: 127
			// (get) Token: 0x06000288 RID: 648 RVA: 0x0000D9B5 File Offset: 0x0000BBB5
			public string Error
			{
				get
				{
					return string.Empty;
				}
			}

			// Token: 0x17000080 RID: 128
			public string this[string columnName]
			{
				get
				{
					string text = string.Empty;
					if (columnName != null && (columnName == "FkMaster" || columnName == "FkDetail") && this.FkMaster.Split(new char[] { ',' }).Count<string>() != this.FkDetail.Split(new char[] { ',' }).Count<string>())
					{
						text = "FKMaster 與 FkDetail 設定數量不同";
					}
					return text;
				}
			}

			// Token: 0x04000164 RID: 356
			private static readonly string ColumnCheckPattern = "[\"']";

			// Token: 0x04000165 RID: 357
			private string _tblName;

			// Token: 0x04000166 RID: 358
			private string _pk;

			// Token: 0x04000167 RID: 359
			private string _parent;

			// Token: 0x04000168 RID: 360
			private string _fkMaster;

			// Token: 0x04000169 RID: 361
			private string _fkDetail = string.Empty;

			// Token: 0x0400016A RID: 362
			private string _upperTable;

			// Token: 0x0400016B RID: 363
			private string _upperKey;

			// Token: 0x0400016C RID: 364
			private string _thisKey;
		}

		// Token: 0x02000043 RID: 67
		private enum MoveType
		{
			// Token: 0x04000170 RID: 368
			Modify,
			// Token: 0x04000171 RID: 369
			Create,
			// Token: 0x04000172 RID: 370
			Null
		}
	}
}
