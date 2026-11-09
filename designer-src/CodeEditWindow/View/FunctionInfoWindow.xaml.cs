using System;
using System.CodeDom.Compiler;
using System.Collections;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Threading;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Helper;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000033 RID: 51
	public partial class FunctionInfoWindow : Window
	{
		// Token: 0x060001F0 RID: 496 RVA: 0x0000F9E8 File Offset: 0x0000DBE8
		private static void OnDataModelPropertyChanged(DependencyObject sender, DependencyPropertyChangedEventArgs e)
		{
			if (!(e.NewValue is AddPointModel))
			{
				return;
			}
			FunctionInfoWindow functionInfoWindow = sender as FunctionInfoWindow;
			if (functionInfoWindow == null)
			{
				return;
			}
			functionInfoWindow.DataContext = functionInfoWindow.DataModel;
			functionInfoWindow.ReferenceName = functionInfoWindow.DataModel.Name;
		}

		// Token: 0x17000049 RID: 73
		// (get) Token: 0x060001F1 RID: 497 RVA: 0x0000FA2E File Offset: 0x0000DC2E
		// (set) Token: 0x060001F2 RID: 498 RVA: 0x0000FA40 File Offset: 0x0000DC40
		public AddPointModel DataModel
		{
			get
			{
				return (AddPointModel)base.GetValue(FunctionInfoWindow.DataModelProperty);
			}
			set
			{
				base.SetValue(FunctionInfoWindow.DataModelProperty, value);
			}
		}

		// Token: 0x1700004A RID: 74
		// (get) Token: 0x060001F3 RID: 499 RVA: 0x0000FA4E File Offset: 0x0000DC4E
		// (set) Token: 0x060001F4 RID: 500 RVA: 0x0000FA56 File Offset: 0x0000DC56
		private string ReferenceName { get; set; }

		// Token: 0x060001F5 RID: 501 RVA: 0x0000FAA4 File Offset: 0x0000DCA4
		public FunctionInfoWindow()
		{
			this.DataModel = new AddPointModel(Application.Current.MainWindow.Tag as PackageKey);
			this.DataModel.Status = Status.CREATE;
			if (SettingManager.Get().GetTzpManger(this.DataModel.ProgramKey).isIndFun)
			{
				this.DataModel.FunctionName = string.Format("{0}_{1}_", this.DataModel.ProgramKey.Program, ResourceController.GetInstance().GetProgramInfo(this.DataModel.ProgramKey).Topind);
			}
			else
			{
				this.DataModel.FunctionName = string.Format("{0}_", this.DataModel.ProgramKey.Program);
			}
			CodeSamepleModel codeSamepleModel = ResourceController.GetInstance().CodeSamples.Where<CodeSamepleModel>((CodeSamepleModel s) => s.ID == "sub_memo").ElementAtOrDefault<CodeSamepleModel>(0);
			if (codeSamepleModel != null)
			{
				this.DataModel.Description = codeSamepleModel.Content;
			}
			else
			{
				this.DataModel.Description = string.Format("#+ ", new object[0]);
			}
			this.InitializeComponent();
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(delegate(PackageKey programName)
			{
				this.Dispose();
			});
			base.CommandBindings.Add(new CommandBinding(FunctionInfoCommands.OKCommand, new ExecutedRoutedEventHandler(this.EnableOKButton), new CanExecuteRoutedEventHandler(this.CanExecuteOK)));
			this.OKButton.Click += this.OKButton_Click;
			this.OKButton.Command = FunctionInfoCommands.OKCommand;
			this.CancelButton.Click += this.CancelButton_Click;
			base.DataContext = this.DataModel;
			base.Dispatcher.BeginInvoke(DispatcherPriority.Background, new Action(delegate
			{
				Keyboard.Focus(this.NameField);
				this.NameField.CaretIndex = this.NameField.Text.Length;
			}));
		}

		// Token: 0x060001F6 RID: 502 RVA: 0x0000FC89 File Offset: 0x0000DE89
		public new void Show()
		{
			base.Owner = Application.Current.MainWindow;
			base.ShowDialog();
		}

		// Token: 0x060001F7 RID: 503 RVA: 0x0000FCA2 File Offset: 0x0000DEA2
		public void Show(AddPointModel model)
		{
			this.DataModel = model;
			this.Show(this.DataModel.Type.ToString(), false);
		}

		// Token: 0x060001F8 RID: 504 RVA: 0x0000FCC7 File Offset: 0x0000DEC7
		public void Show(string funcType)
		{
			this.Show(funcType, true);
		}

		// Token: 0x060001F9 RID: 505 RVA: 0x0000FCD4 File Offset: 0x0000DED4
		private void Show(string funcType, bool useDefaultScope)
		{
			base.DataContext = null;
			if (funcType.Equals(DefinitionType.DIALOG.ToString(), StringComparison.InvariantCulture))
			{
				this.TypeField.Items.Clear();
				this.TypeField.Items.Add(DefinitionType.DIALOG.ToString());
				this.DataModel.Type = DefinitionType.DIALOG;
			}
			else if (funcType.Equals(DefinitionType.REPORT.ToString(), StringComparison.InvariantCulture))
			{
				this.TypeField.Items.Clear();
				this.TypeField.Items.Add(DefinitionType.REPORT.ToString());
				this.DataModel.Type = DefinitionType.REPORT;
			}
			else if (funcType.Equals(DefinitionType.FUNCTION.ToString(), StringComparison.InvariantCulture))
			{
				this.TypeField.Items.Clear();
				this.TypeField.Items.Add(DefinitionType.FUNCTION.ToString());
				this.DataModel.Type = DefinitionType.FUNCTION;
			}
			this.UsageField.IsEnabled = true;
			string value = ResourceController.GetInstance().GetProgramInfo(this.DataModel.ProgramKey).TAP.Attribute("type").Value;
			string text;
			if (value.ToUpper() == "B")
			{
				this.DataModel.Scope = Scope.PUBLIC;
				this.UsageField.IsEnabled = false;
			}
			else if (ResourceController.GetInstance().GetProgramInfo(this.DataModel.ProgramKey).Type != null && useDefaultScope && (text = value.ToUpper()) != null)
			{
				if (<PrivateImplementationDetails>{445C380A-FFF5-4AE9-BE74-AFF3AEFBB769}.$$method0x60001dd-1 == null)
				{
					<PrivateImplementationDetails>{445C380A-FFF5-4AE9-BE74-AFF3AEFBB769}.$$method0x60001dd-1 = new Dictionary<string, int>(8)
					{
						{ "M", 0 },
						{ "G", 1 },
						{ "X", 2 },
						{ "Z", 3 },
						{ "Q", 4 },
						{ "S", 5 },
						{ "B", 6 },
						{ "W", 7 }
					};
				}
				int num;
				if (<PrivateImplementationDetails>{445C380A-FFF5-4AE9-BE74-AFF3AEFBB769}.$$method0x60001dd-1.TryGetValue(text, out num))
				{
					switch (num)
					{
					case 0:
					case 1:
					case 2:
					case 3:
					case 4:
						this.DataModel.Scope = Scope.PRIVATE;
						this.UsageField.IsEnabled = false;
						break;
					case 5:
						this.DataModel.Scope = Scope.PRIVATE;
						break;
					case 6:
						this.DataModel.Scope = Scope.PUBLIC;
						this.UsageField.IsEnabled = false;
						break;
					case 7:
						this.DataModel.Scope = Scope.PUBLIC;
						break;
					}
				}
			}
			base.DataContext = this.DataModel;
			this.Show();
		}

		// Token: 0x060001FA RID: 506 RVA: 0x0000FF7E File Offset: 0x0000E17E
		private void OKButton_Click(object sender, RoutedEventArgs e)
		{
			e.Handled = true;
			this.FinishWork();
		}

		// Token: 0x060001FB RID: 507 RVA: 0x0000FF90 File Offset: 0x0000E190
		private void FinishWork()
		{
			Status status = this.DataModel.Status;
			if (status == Status.CREATE)
			{
				if (this.DataModel.IsLoaded)
				{
					this.ModifyModel();
				}
				else
				{
					this.CreateModel();
				}
			}
			else
			{
				this.ModifyModel();
			}
			if (!base.IsActive)
			{
				return;
			}
			base.DialogResult = new bool?(true);
			this.Dispose();
			base.Close();
		}

		// Token: 0x060001FC RID: 508 RVA: 0x00010014 File Offset: 0x0000E214
		private void ModifyModel()
		{
			ModifyInfomation modifyInfomation = new ModifyInfomation();
			modifyInfomation.ProgramKey = this.DataModel.ProgramKey;
			modifyInfomation.Source = this.ReferenceName;
			modifyInfomation.Modified = this.DataModel.FunctionName;
			AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.DataModel.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == this.ReferenceName && (a.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
			if (addPointModel.Description != this.DataModel.Description)
			{
				addPointModel.Description = this.DataModel.Description;
				modifyInfomation.ModifyType = ModifyInfomation.ModifyTypeEnum.Other;
			}
			if (addPointModel.Scope != this.DataModel.Scope)
			{
				addPointModel.Scope = this.DataModel.Scope;
				modifyInfomation.ModifyType = ModifyInfomation.ModifyTypeEnum.Other;
			}
			if (addPointModel.Name != this.DataModel.Name)
			{
				modifyInfomation.ModifyType |= ModifyInfomation.ModifyTypeEnum.Name;
			}
			ResourceController.GetInstance().GetProgramInfo(this.DataModel.ProgramKey).Modify(modifyInfomation);
			EventController.GetInstance().GetEvent<ModifyFunctionEvent>().Publish(modifyInfomation);
		}

		// Token: 0x060001FD RID: 509 RVA: 0x00010134 File Offset: 0x0000E334
		private void CreateModel()
		{
			CreateFunctionContentArgs createFunctionContentArgs = new CreateFunctionContentArgs();
			createFunctionContentArgs.Content = this.DataModel.ToString();
			createFunctionContentArgs.ProgramKey = this.DataModel.ProgramKey;
			EventController.GetInstance().GetEvent<CreateFunctionEvent>().Publish(createFunctionContentArgs);
		}

		// Token: 0x060001FE RID: 510 RVA: 0x00010179 File Offset: 0x0000E379
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(false);
			this.Dispose();
			base.Close();
		}

		// Token: 0x060001FF RID: 511 RVA: 0x00010194 File Offset: 0x0000E394
		public void Dispose()
		{
			BindingOperations.ClearAllBindings(this);
			base.CommandBindings.Clear();
			this.OKButton.Click -= this.OKButton_Click;
			this.OKButton.CommandBindings.Clear();
			this.CancelButton.Click -= this.CancelButton_Click;
		}

		// Token: 0x06000200 RID: 512 RVA: 0x000101F0 File Offset: 0x0000E3F0
		private void EnableOKButton(object sender, ExecutedRoutedEventArgs e)
		{
			this.OKButton.IsEnabled = true;
		}

		// Token: 0x06000201 RID: 513 RVA: 0x000101FE File Offset: 0x0000E3FE
		private void CanExecuteOK(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = FunctionInfoWindow.IsValid(sender as DependencyObject);
		}

		// Token: 0x06000202 RID: 514 RVA: 0x00010214 File Offset: 0x0000E414
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
					if (!FunctionInfoWindow.IsValid(dependencyObject))
					{
						flag = false;
					}
				}
			}
			return flag;
		}

		// Token: 0x040000D0 RID: 208
		public static readonly DependencyProperty DataModelProperty = DependencyProperty.Register("DataModel", typeof(AddPointModel), typeof(FunctionInfoWindow), new FrameworkPropertyMetadata(null, new PropertyChangedCallback(FunctionInfoWindow.OnDataModelPropertyChanged)));
	}
}
